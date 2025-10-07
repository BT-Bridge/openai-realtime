// tools/audio.go
package tools

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/bt-bridge/openai-realtime/shared"
	"github.com/ebitengine/oto/v3"
	"github.com/hraban/opus"
	"github.com/pion/mediadevices"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"go.uber.org/zap"
)

type AudioBuffer struct {
	buffer []byte
	mu     sync.Mutex
	cond   *sync.Cond
	size   int // current bytes in buffer
	cap    int // max bytes the buffer can hold before dropping from head
}

func NewAudioBuffer(capacityBytes int) *AudioBuffer {
	if capacityBytes < 4096 {
		capacityBytes = 4096
	}
	ab := &AudioBuffer{
		buffer: make([]byte, 0, capacityBytes),
		size:   0,
		cap:    capacityBytes,
	}
	ab.cond = sync.NewCond(&ab.mu)
	return ab
}

func (ab *AudioBuffer) Write(data []byte) (dropped int) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	if ab.size+len(data) > ab.cap {
		drop := ab.size + len(data) - ab.cap
		ab.buffer = ab.buffer[drop:]
		ab.size -= drop
		dropped = drop
	}
	ab.buffer = append(ab.buffer, data...)
	ab.size += len(data)
	ab.cond.Signal()
	return dropped
}

func (ab *AudioBuffer) Read(p []byte) (n int, err error) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	for ab.size == 0 {
		ab.cond.Wait()
	}
	n = copy(p, ab.buffer)
	ab.buffer = ab.buffer[n:]
	ab.size -= n
	return n, nil
}

func StreamLocalAudio(ctx context.Context, logger shared.LoggerAdapter, track *webrtc.TrackLocalStaticSample, mediaTrack mediadevices.Track, frameDuration time.Duration) {
	reader, err := mediaTrack.NewEncodedReader(track.Codec().MimeType)
	if err != nil {
		logger.Error("creating media track reader", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		buf, release, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				release()
				return
			}
			logger.Error("reading from media track", err)
			time.Sleep(10 * time.Millisecond)
			release()
			continue
		}
		if buf.Samples == 0 {
			release()
			continue
		}
		err = track.WriteSample(media.Sample{
			Data:     buf.Data[:],
			Duration: frameDuration,
		})
		release()
		if err != nil {
			logger.Error("failed to write sample to track", err)
			continue
		}
	}
}

func PlayRemoteAudio(ctx context.Context, logger shared.LoggerAdapter, track *webrtc.TrackRemote, otoBufferMs int) {
	codec := track.Codec()
	sampleRate := int(codec.ClockRate)
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	channels := int(codec.Channels)
	if channels <= 0 {
		channels = 1
	}
	logger.Info("configuring remote audio",
		zap.Int("sample_rate", sampleRate),
		zap.Int("channels", channels),
	)

	// Opus decoder
	decoder, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		logger.Error("creating Opus decoder", err)
		return
	}

	// Oto playback context
	if otoBufferMs <= 0 {
		otoBufferMs = 100
	}
	otoCtx, ready, err := oto.NewContext(
		&oto.NewContextOptions{
			SampleRate:   sampleRate,
			ChannelCount: channels,
			Format:       oto.FormatSignedInt16LE,
			BufferSize:   time.Duration(otoBufferMs) * time.Millisecond,
		},
	)
	if err != nil {
		fmt.Printf("Oto context failed: %v\n", err)
		return
	}
	<-ready

	// Large ring buffer (~5 seconds of audio) to avoid drops under bursty RTP
	ringCapBytes := sampleRate * channels * 2 /*bytes*/ * 5 /*seconds*/
	audioBuffer := NewAudioBuffer(ringCapBytes)
	

	// Pre-allocate PCM buffer for up to 120ms per Opus packet
	// 120ms = sampleRate * 0.12 samples per channel
	maxFrameSamplesPerChannel := max(int(0.12 * float64(sampleRate)), 960) // never below a typical 20ms @ 48k
	pcm := make([]int16, maxFrameSamplesPerChannel*channels)

	// Tiny jitter buffer to smooth bursts (store a couple of 20ms frames)
	jitter := make([][]byte, 0, 5) // ~100ms capacity
	minJitterFrames := 2

	player := otoCtx.NewPlayer(audioBuffer)
	player.Play()
	defer player.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		rtp, _, err := track.ReadRTP()
		if err != nil {
			if err != io.EOF {
				logger.Error("reading RTP packet", err)
			}
			return
		}
		if len(rtp.Payload) == 0 {
			continue
		}

		// Decode Opus -> PCM int16
		n, err := decoder.Decode(rtp.Payload, pcm)
		if err != nil {
			logger.Error("decoding Opus", err)
			continue
		}
		if n <= 0 {
			continue
		}

		// Convert int16 PCM to little-endian bytes (interleaved)
		byteCount := n * channels * 2
		frame := make([]byte, byteCount)
		for i := 0; i < n*channels; i++ {
			binary.LittleEndian.PutUint16(frame[i*2:], uint16(pcm[i]))
		}

		// Jitter accumulation; flush once we have a couple of frames
		jitter = append(jitter, frame)
		if len(jitter) >= minJitterFrames {
			for _, f := range jitter {
				dropped := audioBuffer.Write(f) // ignore "dropped" flag to avoid log spam
				if dropped > 0 {
					logger.Warn("audio buffer overflow; dropping old audio", zap.Int("dropped_bytes", dropped))
				}
			}
			jitter = jitter[:0]
		}
	}
}
