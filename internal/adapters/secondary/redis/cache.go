// Package redis — вторичный адаптер для кеширования результатов анализа в Redis.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/grigory/gomodinspect/internal/core/domain"
)

const cacheTTL = 24 * time.Hour

type Cache struct {
	client *redis.Client
	logger *slog.Logger
}

func NewCache(client *redis.Client, logger *slog.Logger) *Cache {
	return &Cache{client: client, logger: logger}
}

// cacheKey формирует ключ кеша по URL репозитория.
func cacheKey(repoURL string) string {
	return fmt.Sprintf("gomodinspect:analysis:%s", repoURL)
}

// Save сохраняет результат анализа в Redis с TTL.
func (c *Cache) Save(ctx context.Context, record *domain.AnalysisRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("сериализация записи: %w", err)
	}

	key := cacheKey(record.RepoURL)
	if err := c.client.Set(ctx, key, data, cacheTTL).Err(); err != nil {
		return fmt.Errorf("запись в Redis: %w", err)
	}

	c.logger.Debug("результат закеширован", slog.String("key", key))
	return nil
}

// Get возвращает закешированный результат анализа или nil, если кеш пуст.
func (c *Cache) Get(ctx context.Context, repoURL string) (*domain.AnalysisRecord, error) {
	key := cacheKey(repoURL)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
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

// Connect создаёт и проверяет подключение к Redis.
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
