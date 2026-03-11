package ports

import (
	"context"

	"github.com/grigory/gomodinspect/internal/core/domain"
)

// AnalysisCache — порт для кеширования результатов анализа.
type AnalysisCache interface {
	// Save сохраняет результат анализа.
	Save(ctx context.Context, record *domain.AnalysisRecord) error
	// Get возвращает закешированный результат анализа для указанного репозитория.
	Get(ctx context.Context, repoURL string) (*domain.AnalysisRecord, error)
}
