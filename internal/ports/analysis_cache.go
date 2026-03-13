package ports

import (
	"context"

	"github.com/grigory/gomodinspect/internal/core/domain"
)

type AnalysisCache interface {
	Save(ctx context.Context, record *domain.AnalysisRecord) error
	Get(ctx context.Context, repoURL string) (*domain.AnalysisRecord, error)
	AllRepoURLs(ctx context.Context) ([]string, error)
	Delete(ctx context.Context, repoURL string) error
}
