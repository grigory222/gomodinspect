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

// Inspector — основной сервис приложения, реализующий инспекцию Go-модулей
type Inspector struct {
	repoFetcher    ports.RepoFetcher
	versionChecker ports.VersionChecker
	cache          ports.AnalysisCache
	workers        int
	logger         *slog.Logger
}

func NewInspector(
	repoFetcher ports.RepoFetcher,
	versionChecker ports.VersionChecker,
	cache ports.AnalysisCache,
	workers int,
	logger *slog.Logger,
) *Inspector {
	return &Inspector{
		repoFetcher:    repoFetcher,
		versionChecker: versionChecker,
		cache:          cache,
		workers:        workers,
		logger:         logger,
	}
}

// Inspect получает go.mod из репозитория, проверяет обновления зависимостей,
// сохраняет результат в Redis и возвращает ModuleInfo
func (i *Inspector) Inspect(ctx context.Context, repoURL string) (*domain.ModuleInfo, error) {
	i.logger.Info("начинаем анализ", slog.String("repo", repoURL))

	// Если --fast-mode, то пробуем получить из кеша
	if i.cache != nil {
		if record, err := i.cache.Get(ctx, repoURL); err != nil {
			i.logger.Warn("не удалось получить данные из кеша", slog.String("error", err.Error()))
		} else if record != nil {
			var deps []domain.Dependency
			if err := json.Unmarshal([]byte(record.DepsJSON), &deps); err == nil {
				i.logger.Info("результат получен из кеша", slog.String("repo", repoURL))
				return &domain.ModuleInfo{
					Name:       record.ModuleName,
					GoVersion:  record.GoVersion,
					Deps:       deps,
					AnalyzedAt: record.AnalyzedAt,
				}, nil
			}
		}
	}

	// Если из кеша не получилось, то "по-честному" подтягиваем с гитхаба
	goModData, err := i.repoFetcher.FetchGoMod(ctx, repoURL)
	if err != nil {
		return nil, fmt.Errorf("получение go.mod: %w", err)
	}

	// Парсим go.mod, извлекаем название модуля и версию go
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

	// Собираем прямые зависимости
	type job struct {
		name    string
		current string
	}
	var direct []job
	for _, req := range modFile.Require {
		if !req.Indirect {
			direct = append(direct, job{req.Mod.Path, req.Mod.Version})
		}
	}

	// Запускаем worker pool
	// 1 зависимость - 1 джоба. Воркеры читают джобы из канала jobs и обрабатывают их параллельно
	// горутины пишут результаты в канал results
	jobs := make(chan job, len(direct))
	results := make(chan domain.Dependency, len(direct))

	for range i.workers {
		go func() {
			for j := range jobs {
				dep := domain.Dependency{Name: j.name, CurrentVersion: j.current}
				latest, err := i.versionChecker.GetLatestVersion(ctx, j.name)
				if err != nil {
					i.logger.Warn("не удалось проверить последнюю версию",
						slog.String("module", j.name),
						slog.String("error", err.Error()),
					)
					dep.LatestVersion = j.current
				} else {
					dep.LatestVersion = latest
					dep.UpdateAvail = latest != j.current
				}
				results <- dep
			}
		}()
	}

	// кладем джобы в канал, их прочитают и выполнят горутины, которые запустили выше
	for _, j := range direct {
		jobs <- j
	}
	close(jobs)

	// читаем из канала, кладем в слайс
	deps := make([]domain.Dependency, 0, len(direct))
	for range direct {
		deps = append(deps, <-results)
	}

	// сформировать результат
	info := &domain.ModuleInfo{
		Name:       moduleName,
		GoVersion:  goVersion,
		Deps:       deps,
		AnalyzedAt: time.Now(),
	}

	// если --fast-mode, то обновить кэш
	if i.cache != nil {
		depsJSON, err := json.Marshal(deps)
		if err != nil {
			return nil, fmt.Errorf("сериализация зависимостей: %w", err)
		}
		record := &domain.AnalysisRecord{
			RepoURL:    repoURL,
			ModuleName: moduleName,
			GoVersion:  goVersion,
			AnalyzedAt: info.AnalyzedAt,
			DepsJSON:   string(depsJSON),
		}
		if err := i.cache.Save(ctx, record); err != nil {
			i.logger.Error("не удалось закешировать результат анализа", slog.String("error", err.Error()))
		} else {
			i.logger.Info("результат анализа закеширован")
		}
	}

	i.logger.Info("анализ завершён",
		slog.String("module", moduleName),
		slog.Int("total_deps", len(deps)),
	)

	return info, nil
}
