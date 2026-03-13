package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/grigory/gomodinspect/internal/ports"
)

// Refresher периодически (раз в 12 часов) обновляет все закешированные репозитории
// refresher работает только при --fast-mode
type Refresher struct {
	inspector ports.Inspector
	cache     ports.AnalysisCache
	interval  time.Duration
	logger    *slog.Logger
}

func NewRefresher(inspector ports.Inspector, cache ports.AnalysisCache, interval time.Duration, logger *slog.Logger) *Refresher {
	return &Refresher{
		inspector: inspector,
		cache:     cache,
		interval:  interval,
		logger:    logger,
	}
}

// Start запускает фоновый цикл обновления кеша и блокируется до отмены контекста
func (r *Refresher) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.refresh(ctx)
		}
	}
}

// refresh - основная функция этого сервиса. Проходится по всем url'ам из кеша, удаляет их и заново анализирует
func (r *Refresher) refresh(ctx context.Context) {
	urls, err := r.cache.AllRepoURLs(ctx)
	if err != nil {
		r.logger.Error("обновление кеша: не удалось получить список репозиториев", slog.String("error", err.Error()))
		return
	}

	r.logger.Info("обновление кеша", slog.Int("repos", len(urls)))
	for _, url := range urls {
		if err := r.cache.Delete(ctx, url); err != nil {
			r.logger.Warn("обновление кеша: не удалось удалить запись",
				slog.String("repo", url),
				slog.String("error", err.Error()),
			)
			continue
		}
		if _, err := r.inspector.Inspect(ctx, url); err != nil {
			r.logger.Warn("обновление кеша: не удалось обновить репозиторий",
				slog.String("repo", url),
				slog.String("error", err.Error()),
			)
		}
	}
}
