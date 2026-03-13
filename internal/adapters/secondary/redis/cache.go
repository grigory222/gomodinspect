package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/grigory/gomodinspect/internal/core/domain"
)

const cacheKeyPrefix = "gomodinspect:analysis"

type Cache struct {
	client *redis.Client
	ttl    time.Duration
	logger *slog.Logger
}

func NewCache(client *redis.Client, ttl time.Duration, logger *slog.Logger) *Cache {
	return &Cache{client: client, ttl: ttl, logger: logger}
}

// cacheKey формирует ключ кеша по URL репозитория
func cacheKey(repoURL string) string {
	return cacheKeyPrefix + ":" + repoURL
}

func (c *Cache) Save(ctx context.Context, record *domain.AnalysisRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("сериализация записи: %w", err)
	}

	key := cacheKey(record.RepoURL)
	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("запись в Redis: %w", err)
	}

	c.logger.Debug("результат закеширован", slog.String("key", key))
	return nil
}

func (c *Cache) Get(ctx context.Context, repoURL string) (*domain.AnalysisRecord, error) {
	key := cacheKey(repoURL)

	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("чтение из Redis: %w", err)
	}

	var record domain.AnalysisRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("десериализация записи: %w", err)
	}

	c.logger.Debug("попадание в кеш", slog.String("key", key))
	return &record, nil
}

func (c *Cache) Delete(ctx context.Context, repoURL string) error {
	if err := c.client.Del(ctx, cacheKey(repoURL)).Err(); err != nil {
		return fmt.Errorf("удаление из Redis: %w", err)
	}
	return nil
}

func (c *Cache) AllRepoURLs(ctx context.Context) ([]string, error) {
	pattern := cacheKeyPrefix + ":*"
	var urls []string
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		urls = append(urls, key[len(cacheKeyPrefix)+1:])
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("scan Redis: %w", err)
	}
	return urls, nil
}

func Connect(ctx context.Context, addr, password string, db int, logger *slog.Logger) (*redis.Client, error) {
	logger.Info("подключаемся к Redis", slog.String("addr", addr))

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("пинг Redis: %w", err)
	}

	logger.Info("подключение к Redis установлено")
	return client, nil
}
