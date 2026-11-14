package media

import (
	"encoding/json"
	"fmt"
	"time"

	domain "github.com/slidebolt/sb-domain"
	messenger "github.com/slidebolt/sb-messenger-sdk"
)

const catalogTimeout = 15 * time.Second

type MediaItem = domain.MediaItem

type CatalogRequest struct {
	ServerID  string `json:"serverID"`
	LibraryID string `json:"libraryID"`
	ParentID  string `json:"parentID,omitempty"`
	ItemID    string `json:"itemID,omitempty"`
	Query     string `json:"query,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Sort      string `json:"sort,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type CatalogPage struct {
	Items      []MediaItem `json:"items"`
	NextCursor string      `json:"nextCursor,omitempty"`
	Total      int         `json:"total,omitempty"`
	Error      string      `json:"error,omitempty"`
}

type ArtworkRequest struct {
	ServerID string `json:"serverID"`
	ItemID   string `json:"itemID"`
	Kind     string `json:"kind"` // thumb, art
}

type ArtworkReply struct {
	ContentType string `json:"contentType,omitempty"`
	Data        []byte `json:"data,omitempty"`
	Error       string `json:"error,omitempty"`
}

type GuideRequest struct {
	ServerID string    `json:"serverID"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Channel  string    `json:"channel,omitempty"`
	Query    string    `json:"query,omitempty"`
	Cursor   string    `json:"cursor,omitempty"`
	Limit    int       `json:"limit,omitempty"`
}

type GuideChannel struct {
	ID    string           `json:"id"`
	State domain.TVChannel `json:"state"`
}

type GuidePage struct {
	Channels   []GuideChannel     `json:"channels,omitempty"`
	Programs   []domain.TVProgram `json:"programs,omitempty"`
	NextCursor string             `json:"nextCursor,omitempty"`
	Error      string             `json:"error,omitempty"`
}

type ResolveRequest struct {
	Source domain.PlayableSource `json:"source"`
}
type ResolveReply struct {
	URL    string `json:"url,omitempty"`
	Kind   string `json:"kind,omitempty"`
	Format string `json:"format,omitempty"`
	Error  string `json:"error,omitempty"`
}

type Provider interface {
	Browse(CatalogRequest) CatalogPage
	Artwork(ArtworkRequest) ArtworkReply
	Guide(GuideRequest) GuidePage
	Resolve(ResolveRequest) ResolveReply
}

func providerSubject(provider, operation string) string {
	return "media.providers." + provider + "." + operation
}

func (c *Client) Browse(provider string, req CatalogRequest) (CatalogPage, error) {
	var reply CatalogPage
	err := c.providerRequest(providerSubject(provider, "catalog"), req, &reply)
	return reply, err
}

func (c *Client) Artwork(provider string, req ArtworkRequest) (ArtworkReply, error) {
	var reply ArtworkReply
	err := c.providerRequest(providerSubject(provider, "artwork"), req, &reply)
	return reply, err
}

func (c *Client) Guide(provider string, req GuideRequest) (GuidePage, error) {
	var reply GuidePage
	err := c.providerRequest(providerSubject(provider, "guide"), req, &reply)
	return reply, err
}
func (c *Client) Resolve(provider string, req ResolveRequest) (ResolveReply, error) {
	var reply ResolveReply
	err := c.providerRequest(providerSubject(provider, "resolve"), req, &reply)
	return reply, err
}

func (c *Client) providerRequest(subject string, req, reply any) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp, err := c.msg.Request(subject, data, catalogTimeout)
	if err != nil {
		return fmt.Errorf("media provider %s: %w", subject, err)
	}
	return json.Unmarshal(resp.Data, reply)
}

// ServeProvider exposes provider-neutral catalog, artwork, and guide handlers.
func ServeProvider(msg messenger.Messenger, id string, p Provider) ([]messenger.Subscription, error) {
	type registration struct {
		operation string
		handle    func([]byte) any
	}
	regs := []registration{
		{"catalog", func(b []byte) any {
			var r CatalogRequest
			if json.Unmarshal(b, &r) != nil {
				return CatalogPage{Error: "invalid request"}
			}
			return p.Browse(r)
		}},
		{"artwork", func(b []byte) any {
			var r ArtworkRequest
			if json.Unmarshal(b, &r) != nil {
				return ArtworkReply{Error: "invalid request"}
			}
			return p.Artwork(r)
		}},
		{"guide", func(b []byte) any {
			var r GuideRequest
			if json.Unmarshal(b, &r) != nil {
				return GuidePage{Error: "invalid request"}
			}
			return p.Guide(r)
		}},
		{"resolve", func(b []byte) any {
			var r ResolveRequest
			if json.Unmarshal(b, &r) != nil {
				return ResolveReply{Error: "invalid request"}
			}
			return p.Resolve(r)
		}},
	}
	var subs []messenger.Subscription
	for _, reg := range regs {
		r := reg
		sub, err := msg.Subscribe(providerSubject(id, r.operation), func(m *messenger.Message) {
			data, _ := json.Marshal(r.handle(m.Data))
			_ = m.Respond(data)
		})
		if err != nil {
			for _, s := range subs {
				_ = s.Unsubscribe()
			}
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}
