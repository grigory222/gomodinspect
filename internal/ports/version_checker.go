package ports

import "context"

// VersionChecker — порт для проверки последних версий Go-модулей.
type VersionChecker interface {
	// GetLatestVersion возвращает последнюю версию указанного модуля.
	GetLatestVersion(ctx context.Context, modulePath string) (string, error)
}
