// Package ports определяет интерфейсы (порты) для получения содержимого репозитория.
package ports

import "context"

// RepoFetcher — интерфейс для получения содержимого go.mod из репозитория.
type RepoFetcher interface {
	// FetchGoMod возвращает содержимое go.mod для указанного URL репозитория.
	FetchGoMod(ctx context.Context, repoURL string) ([]byte, error)
}
