package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pabloju2003/url-shortener/internal/handler"
	"github.com/pabloju2003/url-shortener/internal/repository"
	"github.com/pabloju2003/url-shortener/internal/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockService implements service.URLService via function fields.
type mockService struct {
	shorten     func(context.Context, string) (*repository.URL, error)
	resolve     func(context.Context, string) (string, error)
	getStats    func(context.Context, string) (*service.StatsResult, error)
	recordClick func(context.Context, int64, string, string) error
}

func (m *mockService) Shorten(ctx context.Context, u string) (*repository.URL, error) {
	return m.shorten(ctx, u)
}
func (m *mockService) Resolve(ctx context.Context, code string) (string, error) {
	return m.resolve(ctx, code)
}
func (m *mockService) GetStats(ctx context.Context, code string) (*service.StatsResult, error) {
	return m.getStats(ctx, code)
}
func (m *mockService) RecordClick(ctx context.Context, urlID int64, ua, ip string) error {
	if m.recordClick != nil {
		return m.recordClick(ctx, urlID, ua, ip)
	}
	return nil
}

func newTestRouter(svc service.URLService) *gin.Engine {
	r := gin.New()
	h := handler.NewURLHandler(svc)
	r.POST("/shorten", h.Shorten)
	r.GET("/stats/:code", h.Stats)
	r.GET("/:code", h.Redirect)
	return r
}

func TestShorten_Success(t *testing.T) {
	svc := &mockService{
		shorten: func(_ context.Context, originalURL string) (*repository.URL, error) {
			return &repository.URL{ID: 1, Code: "abc123", OriginalURL: originalURL}, nil
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/shorten",
		bytes.NewBufferString(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["code"] != "abc123" {
		t.Errorf("code: want %q, got %q", "abc123", resp["code"])
	}
	if resp["original_url"] != "https://example.com" {
		t.Errorf("original_url: want %q, got %q", "https://example.com", resp["original_url"])
	}
}

func TestShorten_InvalidURL(t *testing.T) {
	// Service must not be called when url field is missing.
	svc := &mockService{}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/shorten",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body)
	}
}

func TestRedirect_Success(t *testing.T) {
	svc := &mockService{
		getStats: func(_ context.Context, code string) (*service.StatsResult, error) {
			return &service.StatsResult{
				URL:          &repository.URL{ID: 1, Code: code, OriginalURL: "https://example.com"},
				TopCountries: map[string]int{},
				TopDevices:   map[string]int{},
			}, nil
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	newTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d: %s", w.Code, w.Body)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com" {
		t.Errorf("Location: want %q, got %q", "https://example.com", loc)
	}
}

func TestRedirect_NotFound(t *testing.T) {
	svc := &mockService{
		getStats: func(_ context.Context, _ string) (*service.StatsResult, error) {
			return nil, errors.New("URL not found")
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notexist", nil)
	newTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body)
	}
}

func TestStats_Success(t *testing.T) {
	svc := &mockService{
		getStats: func(_ context.Context, code string) (*service.StatsResult, error) {
			return &service.StatsResult{
				URL:          &repository.URL{ID: 1, Code: code, OriginalURL: "https://example.com"},
				Clicks:       []repository.Click{{ID: 1, URLID: 1, Device: "desktop"}},
				TotalClicks:  1,
				TopCountries: map[string]int{},
				TopDevices:   map[string]int{"desktop": 1},
			}, nil
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stats/abc123", nil)
	newTestRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["TotalClicks"] != float64(1) {
		t.Errorf("TotalClicks: want 1, got %v", resp["TotalClicks"])
	}
	if devices, ok := resp["TopDevices"].(map[string]any); !ok || devices["desktop"] != float64(1) {
		t.Errorf("TopDevices.desktop: want 1, got %v", resp["TopDevices"])
	}
}
