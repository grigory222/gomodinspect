package ports

import (
	"context"

	"github.com/grigory/gomodinspect/internal/core/domain"
)

// Inspector — интерфейс для инспекции Go-модуля репозитория (входящий порт / use case).
type Inspector interface {
	Inspect(ctx context.Context, repoURL string) (*domain.ModuleInfo, error)
}
