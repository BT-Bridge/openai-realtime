package realtime

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"sync"
	"time"

	"github.com/bt-bridge/openai-realtime/shared"
	"github.com/openai/openai-go/v3/realtime"
	"github.com/pion/webrtc/v4"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type TrackRemoteHandler func(track *webrtc.TrackRemote)
type TrackLocalHandler func(track *webrtc.TrackLocalStaticSample)

type EventHandler func(event *Event)

type Client struct {
	logger  shared.LoggerAdapter
	baseUrl *url.URL
	apiKey  string
	cfg     *realtime.RealtimeSessionCreateRequestParam

	mu      sync.Mutex
	pc      *webrtc.PeerConnection
	dc      *webrtc.DataChannel
	running bool

	audioL   *webrtc.TrackLocalStaticSample
	audioTLH TrackLocalHandler  // track.Kind() == webrtc.RTPCodecTypeAudio
	audioTRH TrackRemoteHandler // track.Kind() == webrtc.RTPCodecTypeAudio
	eh       EventHandler
}

func NewClient(logger shared.LoggerAdapter, apikey string, baseUrl string) (c *Client, err error) {
	if logger == nil {
		return nil, shared.ErrNoLogger
	}
	if apikey == "" {
		return nil, shared.ErrNoAPIKey
	}
	var baseUrl_ *url.URL
	if baseUrl != "" {
		baseUrl_, err = url.Parse(baseUrl)
		if err != nil {
			return nil, fmt.Errorf("parsing base URL: %w", err)
		}
	} else {
		baseUrl_ = &url.URL{
			Scheme: "https",
			Host:   "api.openai.com",
			Path:   "/v1",
		}
	}
	c = &Client{
		logger:  logger,
		baseUrl: baseUrl_,
		apiKey:  apikey,
	}

	// Creating a new WebRTC API object
	c.pc, err = webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, fmt.Errorf("creating peer connection: %w", err)
	}

	c.dc, err = c.pc.CreateDataChannel("oai", nil)
	if err != nil {
		return nil, fmt.Errorf("creating data channel: %w", err)
	}
	return
}

func (c *Client) SetConfig(cfg *realtime.RealtimeSessionCreateRequestParam) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return shared.ErrSessionAlreadyRunning
	}
	c.cfg = cfg
	return nil
}

func (c *Client) RegisterTrackLocalHandler(handler TrackLocalHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return shared.ErrSessionAlreadyRunning
	}
	if c.audioTLH != nil || c.audioL != nil {
		return shared.ErrTLHandlerAlreadySet
	}
	if handler == nil {
		return errors.New("handler is required")
	}
	// Setting audio track
	var err error
	c.audioL, err = webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeOpus,
			ClockRate:    48000,
			Channels:     2,
			SDPFmtpLine:  "minptime=10;useinbandfec=1",
			RTCPFeedback: nil,
		},
		"audio",
		"mic",
	)
	if err != nil {
		return fmt.Errorf("creating local audio track: %w", err)
	}
	_, err = c.pc.AddTrack(c.audioL)
	if err != nil {
		return fmt.Errorf("adding audio track to peer connection: %w", err)
	}
	c.audioTLH = handler
	return nil
}

func (c *Client) RegisterTrackRemoteHandler(handler TrackRemoteHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return shared.ErrSessionAlreadyRunning
	}
	if c.audioTRH != nil {
		return shared.ErrTRHandlerAlreadySet
	}
	if handler == nil {
		return errors.New("handler is required")
	}
	c.audioTRH = handler
	c.pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		if track.Kind() == webrtc.RTPCodecTypeAudio {
			go c.audioTRH(track)
		}
	})
	return nil
}

func (c *Client) RegisterEventHandler(handler EventHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return shared.ErrSessionAlreadyRunning
	}
	if c.eh != nil {
		return shared.ErrEHandlerAlreadySet
	}
	if handler == nil {
		return errors.New("handler is required")
	}
	c.eh = handler
	c.dc.OnOpen(func() {
		c.logger.Info("data channel opened")
	})
	c.dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if !msg.IsString {
			c.logger.Warn("received non-string message on data channel")
			return
		}
		event := new(Event)
		if err := event.UnmarshalJSON(msg.Data); err != nil {
			c.logger.Error(
				"can not unmarshal event",
				err,
				zap.ByteString("data", msg.Data),
			)
			return
		}
		c.logger.Info(
			"received event",
			zap.String("type", string(event.Type)),
			zap.String("event_id", event.EventId),
			zap.Any("param", event.Param),
		)
		c.eh(event)
	})
	return nil
}

func (c *Client) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return shared.ErrSessionAlreadyRunning
	}
	if c.cfg == nil {
		return shared.ErrNoConfig
	}
	if c.pc == nil || c.dc == nil {
		return shared.ErrClientNotInitialized
	}
	if c.eh == nil {
		return shared.ErrNoEventHandler
	}
	if c.audioTLH != nil && c.audioL == nil {
		return errors.New("audio track local handler is set but audio track is nil")
	}
	if c.audioTRH != nil && c.audioL == nil {
		return errors.New("audio track remote handler is set but audio track is nil")
	}

	offer, err := c.pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("creating offer: %w", err)
	}
	if err = c.pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("setting local description: %w", err)
	}
	answerOffer, err := c.createSession(offer.SDP)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	if err := c.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answerOffer,
	}); err != nil {
		return err
	}
	connected := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			<-ticker.C
			if c.pc.ConnectionState() == webrtc.PeerConnectionStateConnected {
				close(connected)
				return
			}
			if c.pc.ConnectionState() == webrtc.PeerConnectionStateFailed {
				c.logger.Error("error on connection to OpenAI", errors.New("peer connection state is failed"))
				return
			}
			if c.pc.ConnectionState() == webrtc.PeerConnectionStateClosed {
				c.logger.Error("error on connection to OpenAI", errors.New("peer connection state is closed"))
				return
			}
		}
	}()
	select {
	case <-connected:
		c.logger.Info("peer connection state is connected")
	case <-time.After(10 * time.Second):
		return errors.New("timeout waiting for peer connection to be connected")
	}
	return nil
}

func (c *Client) createSession(offer string) (answerOffer string, err error) {
	sessBytes, err := c.cfg.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// SDP part
	sdpHeaders := textproto.MIMEHeader{}
	sdpHeaders.Set("Content-Disposition", `form-data; name="sdp"`)
	sdpHeaders.Set("Content-Type", "application/sdp")
	sdpPart, err := writer.CreatePart(sdpHeaders)
	if err != nil {
		return "", fmt.Errorf("creating SDP part: %w", err)
	}
	if _, err = sdpPart.Write([]byte(offer)); err != nil {
		return "", fmt.Errorf("writing SDP part: %w", err)
	}

	// Session part
	sessionHeaders := textproto.MIMEHeader{}
	sessionHeaders.Set("Content-Disposition", `form-data; name="session"`)
	sessionHeaders.Set("Content-Type", "application/json")
	sessionPart, err := writer.CreatePart(sessionHeaders)
	if err != nil {
		return "", fmt.Errorf("creating session part: %w", err)
	}
	if _, err = sessionPart.Write(sessBytes); err != nil {
		return "", fmt.Errorf("writing session part: %w", err)
	}

	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("closing multipart writer: %w", err)
	}

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(c.baseUrl.JoinPath("/realtime/calls").String())
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetBody(body.Bytes())

	if err = fasthttp.Do(req, resp); err != nil {
		return "", fmt.Errorf("performing HTTP request: %w", err)
	}
	if resp.StatusCode() != fasthttp.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}
	return string(resp.Body()[:]), nil
}
