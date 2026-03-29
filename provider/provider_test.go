package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ContinuumApp/continuum-plugin-tvdb/metadata"
)

func TestSearchByTitlePrefersThumbnail(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data":   map[string]any{"token": "test-token"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/search":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data": []map[string]any{
					{
						"tvdb_id":   "81189",
						"name":      "Breaking Bad",
						"year":      "2008",
						"image_url": "https://artworks.example/poster-original.jpg",
						"thumbnail": "https://artworks.example/poster-thumb.jpg",
						"overview":  "A chemistry teacher turned drug lord.",
					},
					{
						"tvdb_id":   "99999",
						"name":      "No Thumb Show",
						"year":      "2020",
						"image_url": "https://artworks.example/no-thumb.jpg",
						"thumbnail": "",
						"overview":  "Show without thumbnail.",
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient("test-key", "test-pin", 1000)
	client.SetBaseURL(server.URL)
	p := NewProviderWithClient(client)

	results, err := p.Search(context.Background(), metadata.SearchQuery{
		Title:       "Breaking Bad",
		ContentType: "series",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	// When thumbnail is present, prefer it.
	if results[0].ImageURL != "https://artworks.example/poster-thumb.jpg" {
		t.Fatalf("result[0].ImageURL = %q, want thumbnail", results[0].ImageURL)
	}
	// When thumbnail is empty, fall back to image_url.
	if results[1].ImageURL != "https://artworks.example/no-thumb.jpg" {
		t.Fatalf("result[1].ImageURL = %q, want original image", results[1].ImageURL)
	}
}

func TestSearchByIDPrefersPosterThumbnail(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data":   map[string]any{"token": "test-token"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/series/81189/extended":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data": map[string]any{
					"id":    81189,
					"name":  "Breaking Bad",
					"year":  "2008",
					"image": "https://artworks.example/poster-original.jpg",
					"artworks": []map[string]any{
						{
							"id":        1,
							"type":      2,
							"image":     "https://artworks.example/poster-original.jpg",
							"thumbnail": "https://artworks.example/poster-thumb.jpg",
							"width":     680,
							"height":    1000,
						},
						{
							"id":        2,
							"type":      3,
							"image":     "https://artworks.example/bg-original.jpg",
							"thumbnail": "https://artworks.example/bg-thumb.jpg",
							"width":     1920,
							"height":    1080,
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient("test-key", "test-pin", 1000)
	client.SetBaseURL(server.URL)
	p := NewProviderWithClient(client)

	results, err := p.Search(context.Background(), metadata.SearchQuery{
		ProviderIDs: map[string]string{"tvdb": "81189"},
		ContentType: "series",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	// Should use the poster artwork thumbnail, not the base image.
	if results[0].ImageURL != "https://artworks.example/poster-thumb.jpg" {
		t.Fatalf("result.ImageURL = %q, want poster thumbnail", results[0].ImageURL)
	}
}

func TestGetImagesPrefersThumbnailAndFallsBackToImage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data": map[string]any{
					"token": "test-token",
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/series/99/extended":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data": map[string]any{
					"id":   99,
					"name": "Series",
					"artworks": []map[string]any{
						{
							"id":        1,
							"type":      2,
							"image":     "https://artworks.example/poster-original.jpg",
							"thumbnail": "https://artworks.example/poster-thumb.jpg",
							"width":     2000,
							"height":    3000,
							"score":     10,
						},
						{
							"id":        2,
							"type":      3,
							"image":     "https://artworks.example/background-original.jpg",
							"thumbnail": "",
							"width":     3840,
							"height":    2160,
							"score":     8,
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient("test-key", "test-pin", 1000)
	client.SetBaseURL(server.URL)
	p := NewProviderWithClient(client)

	images, err := p.GetImages(context.Background(), metadata.ImageRequest{
		ProviderIDs: map[string]string{"tvdb": "99"},
		ContentType: "series",
	})
	if err != nil {
		t.Fatalf("GetImages() error = %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("len(images) = %d, want 2", len(images))
	}

	got := map[metadata.ImageType]string{}
	for _, img := range images {
		got[img.Type] = img.URL
	}

	if got[metadata.ImagePoster] != "https://artworks.example/poster-thumb.jpg" {
		t.Fatalf("poster URL = %q", got[metadata.ImagePoster])
	}
	if got[metadata.ImageBackdrop] != "https://artworks.example/background-original.jpg" {
		t.Fatalf("backdrop URL = %q", got[metadata.ImageBackdrop])
	}
}
