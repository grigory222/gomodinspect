// Package services содержит основную бизнес-логику приложения.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/mod/modfile"

	"github.com/grigory/gomodinspect/internal/core/domain"
	"github.com/grigory/gomodinspect/internal/ports"
)

// Inspector — основной сервис приложения, реализующий инспекцию Go-модулей.
type Inspector struct {
	repoFetcher    ports.RepoFetcher
	versionChecker ports.VersionChecker
	cache          ports.AnalysisCache
	logger         *slog.Logger
}

// NewInspector создаёт новый экземпляр сервиса Inspector.
func NewInspector(
	repoFetcher ports.RepoFetcher,
	versionChecker ports.VersionChecker,
	cache ports.AnalysisCache,
	logger *slog.Logger,
) *Inspector {
	return &Inspector{
		repoFetcher:    repoFetcher,
		versionChecker: versionChecker,
		cache:          cache,
		logger:         logger,
	}
}

// Inspect получает go.mod из репозитория, проверяет обновления зависимостей,
// сохраняет результат в БД и возвращает ModuleInfo.
func (i *Inspector) Inspect(ctx context.Context, repoURL string) (*domain.ModuleInfo, error) {
	i.logger.Info("начинаем анализ", slog.String("repo", repoURL))

	// 1. Получаем go.mod
	goModData, err := i.repoFetcher.FetchGoMod(ctx, repoURL)
	if err != nil {
		return nil, fmt.Errorf("получение go.mod: %w", err)
	}

	// 2. Парсим go.mod
	modFile, err := modfile.Parse("go.mod", goModData, nil)
	if err != nil {
		return nil, fmt.Errorf("парсинг go.mod: %w", err)
	}

	moduleName := modFile.Module.Mod.Path
	goVersion := modFile.Go.Version

	i.logger.Info("go.mod разобран",
		slog.String("module", moduleName),
		slog.String("go_version", goVersion),
		slog.Int("deps_count", len(modFile.Require)),
	)

	// 3. Проверяем каждую зависимость на наличие обновлений
	var deps []domain.Dependency
	for _, req := range modFile.Require {
		if req.Indirect {
			continue
		}

		dep := domain.Dependency{
			Name:           req.Mod.Path,
			CurrentVersion: req.Mod.Version,
		}

		latest, err := i.versionChecker.GetLatestVersion(ctx, req.Mod.Path)
		if err != nil {
			i.logger.Warn("не удалось проверить последнюю версию",
				slog.String("module", req.Mod.Path),
				slog.String("error", err.Error()),
			)
			dep.LatestVersion = dep.CurrentVersion
		} else {
			dep.LatestVersion = latest
			dep.UpdateAvail = latest != dep.CurrentVersion
		}

		deps = append(deps, dep)
	}

	info := &domain.ModuleInfo{
		Name:      moduleName,
		GoVersion: goVersion,
		Deps:      deps,
	}

	// 4. Кешируем результат анализа
	depsJSON, err := json.Marshal(deps)
	if err != nil {
		return nil, fmt.Errorf("сериализация зависимостей: %w", err)
	}

	record := &domain.AnalysisRecord{
		RepoURL:    repoURL,
		ModuleName: moduleName,
		GoVersion:  goVersion,
		AnalyzedAt: time.Now().UTC(),
		DepsJSON:   string(depsJSON),
	}

	if err := i.cache.Save(ctx, record); err != nil {
		i.logger.Error("не удалось закешировать результат анализа", slog.String("error", err.Error()))
		// Не фатально: возвращаем результат в любом случае
	} else {
		i.logger.Info("результат анализа закеширован")
	}

	updatable := 0
	for _, d := range deps {
		if d.UpdateAvail {
			updatable++
		}
	}
	i.logger.Info("анализ завершён",
		slog.String("module", moduleName),
		slog.Int("total_deps", len(deps)),
		slog.Int("updatable", updatable),
	)

	return info, nil
}
