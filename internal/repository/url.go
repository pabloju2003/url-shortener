package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type URL struct {
	ID          int64
	Code        string
	OriginalURL string
	CreatedAt   time.Time
}

type Click struct {
	ID        int64
	URLID     int64
	ClickedAt time.Time
	Country   string
	Device    string
}

type URLRepository interface {
	Create(ctx context.Context, originalURL string, code string) (*URL, error)
	GetByCode(ctx context.Context, code string) (*URL, error)
	GetStats(ctx context.Context, urlID int64) ([]Click, error)
	SaveClick(ctx context.Context, urlID int64, country string, device string) error
}

// PostgresURLRepository

type PostgresURLRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresURLRepository(pool *pgxpool.Pool) URLRepository {
	return &PostgresURLRepository{pool: pool}
}

func (r *PostgresURLRepository) Create(ctx context.Context, originalURL string, code string) (*URL, error) {
	const q = `
		INSERT INTO urls (code, original_url)
		VALUES ($1, $2)
		RETURNING id, code, original_url, created_at`

	var u URL
	err := r.pool.QueryRow(ctx, q, code, originalURL).Scan(&u.ID, &u.Code, &u.OriginalURL, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository.Create: %w", err)
	}
	return &u, nil
}

func (r *PostgresURLRepository) GetByCode(ctx context.Context, code string) (*URL, error) {
	const q = `SELECT id, code, original_url, created_at FROM urls WHERE code = $1`

	var u URL
	err := r.pool.QueryRow(ctx, q, code).Scan(&u.ID, &u.Code, &u.OriginalURL, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository.GetByCode: %w", err)
	}
	return &u, nil
}

func (r *PostgresURLRepository) GetStats(ctx context.Context, urlID int64) ([]Click, error) {
	const q = `
		SELECT id, url_id, clicked_at, country, device
		FROM clicks
		WHERE url_id = $1
		ORDER BY clicked_at DESC
		LIMIT 1000`

	rows, err := r.pool.Query(ctx, q, urlID)
	if err != nil {
		return nil, fmt.Errorf("repository.GetStats: %w", err)
	}
	defer rows.Close()

	var clicks []Click
	for rows.Next() {
		var c Click
		if err := rows.Scan(&c.ID, &c.URLID, &c.ClickedAt, &c.Country, &c.Device); err != nil {
			return nil, fmt.Errorf("repository.GetStats scan: %w", err)
		}
		clicks = append(clicks, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.GetStats rows: %w", err)
	}
	return clicks, nil
}

func (r *PostgresURLRepository) SaveClick(ctx context.Context, urlID int64, country string, device string) error {
	const q = `INSERT INTO clicks (url_id, country, device) VALUES ($1, $2, $3)`

	if _, err := r.pool.Exec(ctx, q, urlID, country, device); err != nil {
		return fmt.Errorf("repository.SaveClick: %w", err)
	}
	return nil
}

// RedisCache

const cacheTTL = 24 * time.Hour

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func cacheKey(code string) string {
	return "url:" + code
}

func (c *RedisCache) Set(ctx context.Context, code string, originalURL string) error {
	return c.client.Set(ctx, cacheKey(code), originalURL, cacheTTL).Err()
}

func (c *RedisCache) Get(ctx context.Context, code string) (string, error) {
	return c.client.Get(ctx, cacheKey(code)).Result()
}

func (c *RedisCache) Delete(ctx context.Context, code string) error {
	return c.client.Del(ctx, cacheKey(code)).Err()
}

// Compile-time check that pgx.ErrNoRows is accessible for callers.
var _ = pgx.ErrNoRows
