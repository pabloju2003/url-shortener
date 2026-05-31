package handler

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/pabloju2003/url-shortener/internal/service"
	"github.com/pabloju2003/url-shortener/pkg/shortener"
)

type URLHandler struct {
	service service.URLService
}

func NewURLHandler(svc service.URLService) *URLHandler {
	return &URLHandler{service: svc}
}

func (h *URLHandler) Shorten(c *gin.Context) {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}

	u, err := h.service.Shorten(c.Request.Context(), req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":         u.Code,
		"short_url":    baseURL + "/" + u.Code,
		"original_url": u.OriginalURL,
	})
}

func (h *URLHandler) Redirect(c *gin.Context) {
	code := c.Param("code")
	if !shortener.IsValid(code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
		return
	}

	// GetStats returns the full URL object (with ID) needed by recordClick.
	stats, err := h.service.GetStats(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	go h.recordClick(c.Copy(), stats.URL.ID, c.GetHeader("User-Agent"), c.ClientIP())

	c.Redirect(http.StatusMovedPermanently, stats.URL.OriginalURL)
}

func (h *URLHandler) recordClick(c *gin.Context, urlID int64, userAgent string, ip string) {
	if err := h.service.RecordClick(context.Background(), urlID, userAgent, ip); err != nil {
		log.Printf("recordClick error for urlID=%d ip=%s: %v", urlID, ip, err)
	}
}

func (h *URLHandler) Stats(c *gin.Context) {
	code := c.Param("code")

	result, err := h.service.GetStats(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}
