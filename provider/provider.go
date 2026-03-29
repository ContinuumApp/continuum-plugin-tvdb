package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ContinuumApp/continuum-plugin-tvdb/metadata"
	"github.com/ContinuumApp/continuum-plugin-tvdb/models"
)

const maxCast = 20

// Provider implements SearchProvider, MetadataProvider, ImageProvider,
// and EpisodeProvider for the TVDB v4 API.
type Provider struct {
	client *Client
}

// NewProvider creates a TVDB provider with the given API key and subscriber pin.
func NewProvider(apiKey, pin string) *Provider {
	return &Provider{client: NewClient(apiKey, pin, 10)}
}

// NewProviderWithClient creates a TVDB provider with a pre-configured client.
func NewProviderWithClient(c *Client) *Provider {
	return &Provider{client: c}
}

func (p *Provider) Slug() string       { return "tvdb" }
func (p *Provider) Name() string       { return "TheTVDB" }
func (p *Provider) ForTypes() []string { return []string{"movie", "series"} }

// ---------------------------------------------------------------------------
// SearchProvider
// ---------------------------------------------------------------------------

func (p *Provider) Search(ctx context.Context, query metadata.SearchQuery) ([]metadata.SearchResult, error) {
	// Direct TVDB ID.
	if tvdbID := query.ProviderIDs["tvdb"]; tvdbID != "" {
		id, err := strconv.Atoi(tvdbID)
		if err != nil {
			return nil, fmt.Errorf("tvdb: invalid TVDB ID %q: %w", tvdbID, err)
		}
		return p.searchByID(ctx, id, query.ContentType)
	}

	// IMDb ID lookup.
	if imdbID := query.ProviderIDs["imdb"]; imdbID != "" {
		id, err := p.findByRemoteID(ctx, imdbID, query.ContentType)
		if err != nil || id == 0 {
			return nil, err
		}
		return p.searchByID(ctx, id, query.ContentType)
	}

	// TMDB ID lookup.
	if tmdbID := query.ProviderIDs["tmdb"]; tmdbID != "" {
		id, err := p.findByRemoteID(ctx, tmdbID, query.ContentType)
		if err != nil || id == 0 {
			return nil, err
		}
		return p.searchByID(ctx, id, query.ContentType)
	}

	// Title search.
	if query.Title != "" {
		return p.searchByTitle(ctx, query)
	}

	return nil, nil
}

func (p *Provider) searchByID(ctx context.Context, id int, contentType string) ([]metadata.SearchResult, error) {
	ids := map[string]string{"tvdb": strconv.Itoa(id)}

	switch contentType {
	case "movie":
		movie, err := p.client.GetMovieExtended(ctx, id)
		if err != nil {
			return nil, err
		}
		fillRemoteIDs(ids, movie.RemoteIDs)
		return []metadata.SearchResult{{
			Name:        movie.Name,
			Year:        extractYear(movie.Year),
			ProviderIDs: ids,
			ImageURL:    movie.Image,
			Provider:    p.Slug(),
		}}, nil
	case "series":
		series, err := p.client.GetSeriesExtended(ctx, id)
		if err != nil {
			return nil, err
		}
		fillRemoteIDs(ids, series.RemoteIDs)
		return []metadata.SearchResult{{
			Name:        series.Name,
			Year:        extractYear(series.Year),
			ProviderIDs: ids,
			ImageURL:    series.Image,
			Overview:    series.Overview,
			Provider:    p.Slug(),
		}}, nil
	}
	return nil, nil
}

func (p *Provider) searchByTitle(ctx context.Context, query metadata.SearchQuery) ([]metadata.SearchResult, error) {
	results, err := p.client.Search(ctx, query.Title, query.ContentType)
	if err != nil {
		return nil, err
	}

	var out []metadata.SearchResult
	for _, r := range results {
		out = append(out, metadata.SearchResult{
			Name:        r.Name,
			Year:        extractYear(r.Year),
			ProviderIDs: map[string]string{"tvdb": r.TVDBID},
			ImageURL:    r.ImageURL,
			Overview:    r.Overview,
			Provider:    p.Slug(),
		})
	}
	return out, nil
}

func (p *Provider) findByRemoteID(ctx context.Context, remoteID, mediaType string) (int, error) {
	results, err := p.client.SearchByRemoteID(ctx, remoteID)
	if err != nil {
		return 0, err
	}
	for _, r := range results {
		switch mediaType {
		case "series":
			if r.Series != nil {
				return r.Series.ID, nil
			}
		case "movie":
			if r.Movie != nil {
				return r.Movie.ID, nil
			}
		}
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// MetadataProvider
// ---------------------------------------------------------------------------

func (p *Provider) GetMetadata(ctx context.Context, req metadata.MetadataRequest) (*metadata.MetadataResult, error) {
	tvdbID := req.ProviderIDs["tvdb"]
	if tvdbID == "" {
		return nil, nil
	}
	id, err := strconv.Atoi(tvdbID)
	if err != nil {
		return nil, fmt.Errorf("tvdb: invalid TVDB ID %q: %w", tvdbID, err)
	}
	switch req.ContentType {
	case "movie":
		return p.getMovieMetadata(ctx, id)
	case "series":
		return p.getSeriesMetadata(ctx, id)
	}
	return nil, nil
}

func (p *Provider) getMovieMetadata(ctx context.Context, id int) (*metadata.MetadataResult, error) {
	movie, err := p.client.GetMovieExtended(ctx, id)
	if err != nil {
		return nil, err
	}

	result := &metadata.MetadataResult{
		HasMetadata:      true,
		Title:            movie.Name,
		OriginalLanguage: metadata.NormalizeOriginalLanguage(movie.OriginalLanguage),
		Overview:         findEnglishOverview(movie.Translations),
		Runtime:          movie.Runtime,
		Year:             extractYear(movie.Year),
		ContentRating:    findContentRating(movie.ContentRatings),
		ProviderIDs:      map[string]string{"tvdb": strconv.Itoa(movie.ID)},
	}

	if movie.FirstRelease != nil && movie.FirstRelease.Date != "" && result.Year == 0 {
		result.Year = extractYear(movie.FirstRelease.Date)
	}

	fillRemoteIDs(result.ProviderIDs, movie.RemoteIDs)

	for _, g := range movie.Genres {
		result.Genres = append(result.Genres, g.Name)
	}
	for _, s := range movie.Studios {
		result.Studios = append(result.Studios, s.Name)
	}
	if movie.OriginalCountry != "" {
		result.Countries = []string{movie.OriginalCountry}
	}

	result.People = convertCharacters(movie.Characters)

	return result, nil
}

func (p *Provider) getSeriesMetadata(ctx context.Context, id int) (*metadata.MetadataResult, error) {
	series, err := p.client.GetSeriesExtended(ctx, id)
	if err != nil {
		return nil, err
	}

	officialCount := 0
	for _, s := range series.Seasons {
		if s.Type.ID == 1 && s.Number > 0 {
			officialCount++
		}
	}

	result := &metadata.MetadataResult{
		HasMetadata:      true,
		Title:            series.Name,
		OriginalLanguage: metadata.NormalizeOriginalLanguage(series.OriginalLanguage),
		Overview:         series.Overview,
		Year:             extractYear(series.Year),
		ContentRating:    findContentRating(series.ContentRatings),
		SeasonCount:      officialCount,
		FirstAirDate:     series.FirstAired,
		LastAirDate:      series.LastAired,
		ProviderIDs:      map[string]string{"tvdb": strconv.Itoa(series.ID)},
	}

	fillRemoteIDs(result.ProviderIDs, series.RemoteIDs)

	if series.OriginalNetwork != nil {
		result.Networks = []string{series.OriginalNetwork.Name}
	}
	for _, g := range series.Genres {
		result.Genres = append(result.Genres, g.Name)
	}
	if series.OriginalCountry != "" {
		result.Countries = []string{series.OriginalCountry}
	}

	result.People = convertCharacters(series.Characters)

	return result, nil
}

// ---------------------------------------------------------------------------
// ImageProvider
// ---------------------------------------------------------------------------

func (p *Provider) GetImages(ctx context.Context, req metadata.ImageRequest) ([]metadata.RemoteImage, error) {
	tvdbID := req.ProviderIDs["tvdb"]
	if tvdbID == "" {
		return nil, nil
	}
	id, err := strconv.Atoi(tvdbID)
	if err != nil {
		return nil, fmt.Errorf("tvdb: invalid TVDB ID: %w", err)
	}

	var artworks []ArtworkRecord
	switch req.ContentType {
	case "movie":
		movie, err := p.client.GetMovieExtended(ctx, id)
		if err != nil {
			return nil, err
		}
		artworks = movie.Artworks
	case "series":
		series, err := p.client.GetSeriesExtended(ctx, id)
		if err != nil {
			return nil, err
		}
		artworks = series.Artworks
	}

	var out []metadata.RemoteImage
	for _, a := range artworks {
		imgType, ok := artworkTypeToImageType(a.Type)
		if !ok {
			continue
		}
		out = append(out, metadata.RemoteImage{
			URL:      tvdbPreviewURL(a.Image, a.Thumbnail),
			Type:     imgType,
			Language: a.Language,
			Width:    a.Width,
			Height:   a.Height,
			Rating:   float64(a.Score),
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// EpisodeProvider
// ---------------------------------------------------------------------------

func (p *Provider) GetSeasons(ctx context.Context, req metadata.SeasonsRequest) ([]metadata.SeasonResult, error) {
	tvdbID := req.ProviderIDs["tvdb"]
	if tvdbID == "" {
		return nil, nil
	}
	id, err := strconv.Atoi(tvdbID)
	if err != nil {
		return nil, fmt.Errorf("tvdb: invalid TVDB ID: %w", err)
	}

	series, err := p.client.GetSeriesExtended(ctx, id)
	if err != nil {
		return nil, err
	}

	var seasons []metadata.SeasonResult
	for _, s := range series.Seasons {
		if s.Type.ID != 1 {
			continue
		}
		seasons = append(seasons, metadata.SeasonResult{
			SeasonNumber: s.Number,
			PosterPath:   s.Image,
		})
	}
	return seasons, nil
}

func (p *Provider) GetEpisodes(ctx context.Context, req metadata.EpisodesRequest) ([]metadata.EpisodeResult, error) {
	tvdbID := req.ProviderIDs["tvdb"]
	if tvdbID == "" {
		return nil, nil
	}
	id, err := strconv.Atoi(tvdbID)
	if err != nil {
		return nil, fmt.Errorf("tvdb: invalid TVDB ID: %w", err)
	}

	// Find the season ID matching the requested season number.
	series, err := p.client.GetSeriesExtended(ctx, id)
	if err != nil {
		return nil, err
	}

	var seasonID int
	for _, s := range series.Seasons {
		if s.Type.ID == 1 && s.Number == req.SeasonNumber {
			seasonID = s.ID
			break
		}
	}
	if seasonID == 0 {
		return nil, nil
	}

	season, err := p.client.GetSeasonExtended(ctx, seasonID)
	if err != nil {
		return nil, err
	}

	var episodes []metadata.EpisodeResult
	for _, ep := range season.Episodes {
		episodes = append(episodes, metadata.EpisodeResult{
			ProviderIDs:   map[string]string{"tvdb": strconv.Itoa(ep.ID)},
			SeasonNumber:  ep.SeasonNumber,
			EpisodeNumber: ep.Number,
			Title:         ep.Name,
			Overview:      ep.Overview,
			Runtime:       ep.Runtime,
			AirDate:       ep.Aired,
			StillPath:     ep.Image,
		})
	}
	return episodes, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func extractYear(yearStr string) int {
	if len(yearStr) < 4 {
		return 0
	}
	y, err := strconv.Atoi(yearStr[:4])
	if err != nil {
		return 0
	}
	return y
}

func findEnglishOverview(td *TranslationData) string {
	if td == nil {
		return ""
	}
	for _, t := range td.OverviewTranslations {
		if t.Language == "eng" {
			return t.Overview
		}
	}
	for _, t := range td.OverviewTranslations {
		if t.IsPrimary {
			return t.Overview
		}
	}
	return ""
}

func findContentRating(ratings []ContentRating) string {
	if len(ratings) == 0 {
		return ""
	}
	for _, r := range ratings {
		if r.Country == "usa" {
			return r.Name
		}
	}
	return ratings[0].Name
}

func tvdbPreviewURL(imageURL, thumbnailURL string) string {
	if thumbnailURL != "" {
		return thumbnailURL
	}
	return imageURL
}

func convertCharacters(chars []Character) []models.ItemPerson {
	var people []models.ItemPerson
	castCount := 0
	for _, ch := range chars {
		var kind models.PersonKind
		switch ch.PeopleType {
		case "Actor":
			if castCount >= maxCast {
				continue
			}
			kind = models.PersonKindActor
			castCount++
		case "Guest Star":
			if castCount >= maxCast {
				continue
			}
			kind = models.PersonKindGuestStar
			castCount++
		case "Director":
			kind = models.PersonKindDirector
		case "Writer":
			kind = models.PersonKindWriter
		case "Producer":
			kind = models.PersonKindProducer
		default:
			continue
		}
		people = append(people, models.ItemPerson{
			Person: models.Person{
				Name:      ch.PersonName,
				TvdbID:    strconv.Itoa(ch.PeopleID),
				PhotoPath: ch.PersonImgURL,
			},
			Kind:      kind,
			Character: ch.Name,
			SortOrder: ch.Sort,
		})
	}
	return people
}

func fillRemoteIDs(ids map[string]string, remoteIDs []RemoteID) {
	for _, r := range remoteIDs {
		switch r.Type {
		case 2: // IMDb
			if r.ID != "" && ids["imdb"] == "" {
				ids["imdb"] = r.ID
			}
		case 12: // TMDB
			if r.ID != "" && ids["tmdb"] == "" {
				ids["tmdb"] = r.ID
			}
		}
	}
}

func artworkTypeToImageType(artType int) (metadata.ImageType, bool) {
	switch artType {
	case 2:
		return metadata.ImagePoster, true
	case 3:
		return metadata.ImageBackdrop, true
	case 22:
		return metadata.ImageLogo, true
	default:
		return 0, false
	}
}
