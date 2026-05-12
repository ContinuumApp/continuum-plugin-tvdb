package main

import (
	"context"
	"testing"
	"time"

	pluginv1 "github.com/ContinuumApp/continuum-plugin-sdk/pkg/pluginproto/continuum/plugin/v1"
	"github.com/ContinuumApp/continuum-plugin-tvdb/metadata"
	"github.com/ContinuumApp/continuum-plugin-tvdb/provider"
)

func assertResolvedImageExpiry(t *testing.T, expiresAt time.Time) {
	t.Helper()
	ttl := time.Until(expiresAt)
	if ttl < 23*time.Hour || ttl > 25*time.Hour {
		t.Fatalf("resolved image expiry TTL = %v, want about 24h", ttl)
	}
}

func TestResolveImageURL(t *testing.T) {
	ms := &metadataServer{}
	tests := []struct {
		name, path, variant, want string
	}{
		{"poster card", "banners/posters/81189-10.jpg", "card", "https://artworks.thetvdb.com/banners/posters/81189-10_t.jpg"},
		{"poster featured", "banners/posters/81189-10.jpg", "featured", "https://artworks.thetvdb.com/banners/posters/81189-10.jpg"},
		{"poster original", "banners/posters/81189-10.jpg", "original", "https://artworks.thetvdb.com/banners/posters/81189-10.jpg"},
		{"empty variant", "banners/posters/81189-10.jpg", "", "https://artworks.thetvdb.com/banners/posters/81189-10.jpg"},
		{"backdrop card", "banners/fanart/81189-5.jpg", "card", "https://artworks.thetvdb.com/banners/fanart/81189-5_t.jpg"},
		{"empty path", "", "card", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ms.ResolveImageURL(context.Background(), &pluginv1.ResolveImageURLRequest{
				Path: tt.path, Variant: tt.variant,
			})
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if resp.GetUrl() != tt.want {
				t.Fatalf("got %q, want %q", resp.GetUrl(), tt.want)
			}
			if resp.GetExpiresAt() == nil {
				t.Fatal("expected expires_at to be set")
			}
			assertResolvedImageExpiry(t, resp.GetExpiresAt().AsTime())
		})
	}
}

func TestResolveImageURLsIncludesExpiryAwareMap(t *testing.T) {
	ms := &metadataServer{}
	resp, err := ms.ResolveImageURLs(context.Background(), &pluginv1.ResolveImageURLsRequest{
		Paths:   []string{"banners/posters/81189-10.jpg"},
		Variant: "card",
	})
	if err != nil {
		t.Fatalf("ResolveImageURLs() error = %v", err)
	}
	const wantURL = "https://artworks.thetvdb.com/banners/posters/81189-10_t.jpg"
	if got := resp.GetUrls()["banners/posters/81189-10.jpg"]; got != wantURL {
		t.Fatalf("legacy URL = %q, want %q", got, wantURL)
	}
	resolved := resp.GetResolvedUrls()["banners/posters/81189-10.jpg"]
	if resolved == nil {
		t.Fatal("expected resolved_urls entry")
	}
	if got := resolved.GetUrl(); got != wantURL {
		t.Fatalf("resolved URL = %q, want %q", got, wantURL)
	}
	if resolved.GetExpiresAt() == nil {
		t.Fatal("expected resolved_urls expires_at to be set")
	}
	assertResolvedImageExpiry(t, resolved.GetExpiresAt().AsTime())
}

func TestRuntimeServerConfigure_NoOp(t *testing.T) {
	server := &runtimeServer{provider: provider.NewProvider()}

	_, err := server.Configure(context.Background(), &pluginv1.ConfigureRequest{})
	if err != nil {
		t.Fatalf("Configure() returned error: %v", err)
	}

	p, err := server.providerForRequest()
	if err != nil {
		t.Fatalf("providerForRequest() returned error: %v", err)
	}
	if p == nil {
		t.Fatal("expected provider to be available")
	}
}

func TestPersonDetailRecordFromResult_CanonicalizesPhotoPath(t *testing.T) {
	record, err := personDetailRecordFromResult(&metadata.PersonDetailResult{
		Name:           "Sigourney Weaver",
		SortName:       "Weaver, Sigourney",
		Bio:            "English biography",
		BirthDate:      "1949-10-08",
		Birthplace:     "New York City, New York, USA",
		PhotoPath:      "https://artworks.thetvdb.com/banners/persons/321.jpg",
		PhotoThumbhash: "thumbhash-123",
		ProviderIDs: map[string]string{
			"tvdb": "321",
			"imdb": "nm0000244",
		},
	})
	if err != nil {
		t.Fatalf("personDetailRecordFromResult() error = %v", err)
	}
	if record.GetPhotoPath() != "tvdb://banners/persons/321.jpg" {
		t.Fatalf("record.PhotoPath = %q, want tvdb canonical path", record.GetPhotoPath())
	}
	if record.GetPhotoThumbhash() != "thumbhash-123" {
		t.Fatalf("record.PhotoThumbhash = %q, want thumbhash-123", record.GetPhotoThumbhash())
	}
	if record.GetProviderIds().AsMap()["tvdb"] != "321" {
		t.Fatalf("record.ProviderIds[tvdb] = %#v", record.GetProviderIds().AsMap()["tvdb"])
	}
}
