package media

import (
	"testing"

	domain "github.com/slidebolt/sb-domain"
)

func TestCatalogContractsCarryOpaqueProviderReferences(t *testing.T) {
	item := MediaItem{Ref: domain.MediaItemRef{Provider: "plex", ServerID: "server", LibraryID: "tv", ItemID: "opaque"}, Kind: "episode", Title: "Pilot", Playable: true}
	if item.Ref.ItemID != "opaque" || !item.Playable {
		t.Fatal("catalog item reference was not retained")
	}
}

func TestProviderSubject(t *testing.T) {
	if got := providerSubject("plugin-plex", "guide"); got != "media.providers.plugin-plex.guide" {
		t.Fatalf("subject=%q", got)
	}
}
