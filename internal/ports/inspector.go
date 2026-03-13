package ports

import (
	"context"

	"github.com/grigory/gomodinspect/internal/core/domain"
)

// Inspector - интерфейс для анализа репозитория. Возвращает ModuleInfo с результатом анализа
type Inspector interface {
	Inspect(ctx context.Context, repoURL string) (*domain.ModuleInfo, error)
}
