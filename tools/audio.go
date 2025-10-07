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
)

type AudioBuffer struct {
	buffer []byte
	mu     sync.Mutex
	cond   *sync.Cond
	size   int
	cap    int
}

func NewAudioBuffer(fixedCap int) *AudioBuffer {
	ab := &AudioBuffer{
		buffer: make([]byte, 0, fixedCap),
		size:   0,
		cap:    fixedCap,
	}
	ab.cond = sync.NewCond(&ab.mu)
	return ab
}

func (ab *AudioBuffer) Write(data []byte) (dropped bool) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	if ab.size+len(data) > ab.cap {
		drop := ab.size + len(data) - ab.cap
		ab.buffer = ab.buffer[drop:]
		ab.size -= drop
		dropped = true
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

func PlayRemoteAudio(ctx context.Context, logger shared.LoggerAdapter, track *webrtc.TrackRemote, bufferMs int) {
	var (
		codec      = track.Codec()
		sampleRate = int(codec.ClockRate)
		channels   = int(codec.Channels)
	)
	decoder, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		logger.Error("creating Opus decoder", err)
		return
	}

	otoCtx, ready, err := oto.NewContext(
		&oto.NewContextOptions{
			SampleRate:   sampleRate,
			ChannelCount: channels,
			Format:       oto.FormatSignedInt16LE,
			BufferSize:   time.Duration(bufferMs) * time.Millisecond,
		},
	)
	if err != nil {
		fmt.Printf("Oto context failed: %v\n", err)
		return
	}
	capBytes := int(float64(bufferMs)/1000*float64(sampleRate)) * channels * 2 // 16-bits
	audioBuffer := NewAudioBuffer(capBytes)
	maxFrameSamplesPerChannel := int(1.2 * float64(sampleRate)) // 120ms max frame size
	pcm := make([]int16, maxFrameSamplesPerChannel*channels)
	<-ready
	otoCtx.NewPlayer(audioBuffer).Play()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			rtp, _, err := track.ReadRTP()
			if err != nil {
				if err != io.EOF {
					logger.Error("reading RTP packet", err)
				}
				return
			}
			if len(rtp.Payload) == 0 {
				logger.Error("empty RTP payload", nil)
				continue
			}
			// Decode Opus to PCM 16-bit
			n, err := decoder.Decode(rtp.Payload, pcm)
			if err != nil {
				logger.Error("decoding Opus", err)
				continue
			}
			pcmSlice := pcm[:n*channels]
			// Convert stereo int16 to bytes (interleaved: L, R)
			pcmBytes := make([]byte, len(pcmSlice)*2)
			for i := 0; i < len(pcmSlice); i++ {
				binary.LittleEndian.PutUint16(pcmBytes[i*2:], uint16(pcmSlice[i]))
			}
			// Write to audioBuffer
			dropped := audioBuffer.Write(pcmBytes)
			if dropped {
				logger.Warn("audio buffer dropped data")
			}
		}
	}
}
