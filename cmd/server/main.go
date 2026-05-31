package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pabloju2003/url-shortener/internal/handler"
	"github.com/pabloju2003/url-shortener/internal/repository"
	"github.com/pabloju2003/url-shortener/internal/service"
	"github.com/pabloju2003/url-shortener/migrations"
	"github.com/redis/go-redis/v9"
)

func runMigrations(db *pgxpool.Pool) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		log.Fatalf("failed to load migration source: %v", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	// golang-migrate pgx/v5 driver expects pgx5:// scheme
	migrateURL := "pgx5://" + dbURL[len("postgres://"):]

	m, err := migrate.NewWithSourceInstance("iofs", src, migrateURL)
	if err != nil {
		log.Fatalf("failed to create migrator: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration failed: %v", err)
	}
	log.Println("Migrations applied successfully")
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("postgres ping failed: %v", err)
	}
	log.Println("connected to postgres")

	runMigrations(db)

	redisURL := os.Getenv("REDIS_URL")
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("failed to parse redis URL: %v", err)
	}
	rdb := redis.NewClient(opts)
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}
	log.Println("connected to redis")

	repo := repository.NewPostgresURLRepository(db)
	cache := repository.NewRedisCache(rdb)
	svc := service.NewURLService(repo, cache)
	h := handler.NewURLHandler(svc)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.POST("/shorten", h.Shorten)
	r.GET("/stats/:code", h.Stats)
	r.GET("/:code", h.Redirect)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server listening on port %s", port)
	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
