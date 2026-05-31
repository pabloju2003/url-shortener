package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/pabloju2003/url-shortener/internal/repository"
	"github.com/pabloju2003/url-shortener/pkg/shortener"
	"github.com/redis/go-redis/v9"
)

type StatsResult struct {
	URL          *repository.URL
	Clicks       []repository.Click
	TotalClicks  int
	TopCountries map[string]int
	TopDevices   map[string]int
}

type URLService interface {
	Shorten(ctx context.Context, originalURL string) (*repository.URL, error)
	Resolve(ctx context.Context, code string) (string, error)
	GetStats(ctx context.Context, code string) (*StatsResult, error)
	RecordClick(ctx context.Context, urlID int64, userAgent string, ip string) error
}

type URLServiceImpl struct {
	repo  repository.URLRepository
	cache *repository.RedisCache
}

func NewURLService(repo repository.URLRepository, cache *repository.RedisCache) URLService {
	return &URLServiceImpl{repo: repo, cache: cache}
}

func (s *URLServiceImpl) Shorten(ctx context.Context, originalURL string) (*repository.URL, error) {
	if originalURL == "" {
		return nil, errors.New("originalURL must not be empty")
	}
	parsed, err := url.ParseRequestURI(originalURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid URL: %q", originalURL)
	}

	code, err := shortener.Generate(6)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	u, err := s.repo.Create(ctx, originalURL, code)
	if err != nil {
		return nil, fmt.Errorf("failed to save URL: %w", err)
	}

	if err := s.cache.Set(ctx, code, originalURL); err != nil {
		// Non-fatal: DB is source of truth.
		_ = err
	}

	return u, nil
}

func (s *URLServiceImpl) Resolve(ctx context.Context, code string) (string, error) {
	originalURL, err := s.cache.Get(ctx, code)
	if err == nil {
		return originalURL, nil
	}
	if !errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("cache error: %w", err)
	}

	u, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return "", errors.New("URL not found")
	}

	_ = s.cache.Set(ctx, code, u.OriginalURL)

	return u.OriginalURL, nil
}

func (s *URLServiceImpl) RecordClick(ctx context.Context, urlID int64, userAgent string, ip string) error {
	device := "desktop"
	if strings.Contains(userAgent, "Mobile") {
		device = "mobile"
	}

	if err := s.repo.SaveClick(ctx, urlID, "", device); err != nil {
		return fmt.Errorf("failed to record click: %w", err)
	}
	return nil
}

func (s *URLServiceImpl) GetStats(ctx context.Context, code string) (*StatsResult, error) {
	u, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, errors.New("URL not found")
	}

	clicks, err := s.repo.GetStats(ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}

	countries := make(map[string]int)
	devices := make(map[string]int)
	for _, c := range clicks {
		if c.Country != "" {
			countries[c.Country]++
		}
		if c.Device != "" {
			devices[c.Device]++
		}
	}

	return &StatsResult{
		URL:          u,
		Clicks:       clicks,
		TotalClicks:  len(clicks),
		TopCountries: countries,
		TopDevices:   devices,
	}, nil
}
