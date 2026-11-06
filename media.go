// Package media defines the contract between SlideBolt plugins and the
// sb-media service. Plugins own their entities and sources. sb-media turns
// incoming media into a URL. Control travels over the messenger, media does not.
package media

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	messenger "github.com/slidebolt/sb-messenger-sdk"
)

// Messenger subjects.
const (
	SubjectRegister       = "media.streams.register"
	SubjectStart          = "media.streams.start"
	SubjectStop           = "media.streams.stop"
	SubjectServiceStarted = "media.service.started"
)

// Stream statuses.
const (
	StatusIdle     = "idle"
	StatusStarting = "starting"
	StatusLive     = "live"
	StatusError    = "error"
)

// Stream kinds. Only KindAudio is implemented.
const (
	KindAudio = "audio"
	KindVideo = "video"
	KindImage = "image"
)

// FormatMPEGTS is the only output format implemented.
const FormatMPEGTS = "mpegts"

// Input types and codecs.
const (
	InputRTP  = "rtp"
	CodecPCMU = "pcmu"
)

const requestTimeout = 5 * time.Second

var streamPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)*$`)

// ValidStream reports whether stream is safe to use in subjects and URLs.
func ValidStream(stream string) bool {
	return len(stream) <= 255 && streamPattern.MatchString(stream)
}

// StatusSubject returns the subject status events for a stream are published on.
func StatusSubject(stream string) string {
	return "media.streams." + stream + ".status"
}

type RegisterRequest struct {
	Stream string `json:"stream"`
	Kind   string `json:"kind"`
	Format string `json:"format"`
}

type RegisterReply struct {
	URL   string `json:"url,omitempty"`
	Error string `json:"error,omitempty"`
}

type Input struct {
	Type       string `json:"type"`
	Codec      string `json:"codec"`
	SampleRate int    `json:"sampleRate"`
}

type StartRequest struct {
	Stream string `json:"stream"`
	Input  Input  `json:"input"`
}

type StartReply struct {
	SessionID string `json:"sessionId,omitempty"`
	RTPHost   string `json:"rtpHost,omitempty"`
	RTPPort   int    `json:"rtpPort,omitempty"`
	Error     string `json:"error,omitempty"`
}

type StopRequest struct {
	SessionID string `json:"sessionId"`
}

type StopReply struct {
	Error string `json:"error,omitempty"`
}

// StatusEvent reports a change in a stream session.
type StatusEvent struct {
	Stream    string `json:"stream"`
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
	Viewers   int    `json:"viewers"`
	Error     string `json:"error,omitempty"`
}

// Client is used by plugins to talk to sb-media.
type Client struct {
	msg messenger.Messenger
}

// NewClient creates a Client on an existing messenger connection.
func NewClient(msg messenger.Messenger) *Client {
	return &Client{msg: msg}
}

// Register declares a stream and returns its stable URL.
func (c *Client) Register(req RegisterRequest) (string, error) {
	var reply RegisterReply
	if err := c.request(SubjectRegister, req, &reply); err != nil {
		return "", err
	}
	if reply.Error != "" {
		return "", errors.New(reply.Error)
	}
	return reply.URL, nil
}

// Start opens a session and returns where to send media.
func (c *Client) Start(req StartRequest) (StartReply, error) {
	var reply StartReply
	if err := c.request(SubjectStart, req, &reply); err != nil {
		return StartReply{}, err
	}
	if reply.Error != "" {
		return StartReply{}, errors.New(reply.Error)
	}
	return reply, nil
}

// Stop ends a session.
func (c *Client) Stop(sessionID string) error {
	var reply StopReply
	if err := c.request(SubjectStop, StopRequest{SessionID: sessionID}, &reply); err != nil {
		return err
	}
	if reply.Error != "" {
		return errors.New(reply.Error)
	}
	return nil
}

// OnStatus subscribes to status events for a stream.
func (c *Client) OnStatus(stream string, handler func(StatusEvent)) (messenger.Subscription, error) {
	return c.msg.Subscribe(StatusSubject(stream), func(m *messenger.Message) {
		var ev StatusEvent
		if err := json.Unmarshal(m.Data, &ev); err == nil {
			handler(ev)
		}
	})
}

// OnServiceStarted is called each time sb-media starts. All sessions are gone
// by then, because sb-manager restarts sb-media without restarting dependents.
func (c *Client) OnServiceStarted(handler func()) (messenger.Subscription, error) {
	return c.msg.Subscribe(SubjectServiceStarted, func(*messenger.Message) { handler() })
}

func (c *Client) request(subject string, req, reply any) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("media: marshal: %w", err)
	}
	resp, err := c.msg.Request(subject, data, requestTimeout)
	if err != nil {
		return fmt.Errorf("media: %s: %w", subject, err)
	}
	if err := json.Unmarshal(resp.Data, reply); err != nil {
		return fmt.Errorf("media: decode %s reply: %w", subject, err)
	}
	return nil
}
