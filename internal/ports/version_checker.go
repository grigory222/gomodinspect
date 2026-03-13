package ports

import "context"

// VersionChecker - интерфейс для получения последних версий
type VersionChecker interface {
	GetLatestVersion(ctx context.Context, modulePath string) (string, error)
}
