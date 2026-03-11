// Точка входа CLI-приложения gomodinspect.
// Анализирует Go-модуль указанного GitHub-репозитория и показывает устаревшие зависимости.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	gh "github.com/google/go-github/v69/github"
	"github.com/joho/godotenv"

	"github.com/grigory/gomodinspect/internal/adapters/primary/cli"
	ghAdapter "github.com/grigory/gomodinspect/internal/adapters/secondary/github"
	"github.com/grigory/gomodinspect/internal/adapters/secondary/goproxy"
	redisAdapter "github.com/grigory/gomodinspect/internal/adapters/secondary/redis"
	"github.com/grigory/gomodinspect/internal/config"
	"github.com/grigory/gomodinspect/internal/core/services"
	"github.com/grigory/gomodinspect/internal/logger"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Использование: gomodinspect <github-repo-url>\n")
		fmt.Fprintf(os.Stderr, "Пример: gomodinspect https://github.com/gin-gonic/gin\n")
		os.Exit(1)
	}

	repoURL := os.Args[1]

	_ = godotenv.Load()

	// Определяем путь к конфигурации
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg := config.MustLoad(cfgPath)

	// Логгер
	log := logger.New(cfg.Log.Level)

	// Graceful shutdown: контекст с отменой по сигналу ОС
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Подключение к Redis
	rdb, err := redisAdapter.Connect(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, log)
	if err != nil {
		log.Error("не удалось подключиться к Redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		log.Info("закрываем соединение с Redis")
		_ = rdb.Close()
	}()

	// GitHub-клиент
	var ghClient *gh.Client
	if cfg.GitHub.Token != "" {
		ghClient = gh.NewClient(nil).WithAuthToken(cfg.GitHub.Token)
	} else {
		ghClient = gh.NewClient(nil)
	}

	// Подключаем адаптеры (secondary)
	repoFetcher := ghAdapter.NewRepoFetcher(ghClient, log)
	versionChecker := goproxy.NewVersionChecker(log)
	cache := redisAdapter.NewCache(rdb, log)

	// Сервис приложения (core)
	inspector := services.NewInspector(repoFetcher, versionChecker, cache, log)

	// CLI-адаптер (primary)
	runner := cli.NewRunner(inspector, log)

	if err := runner.Run(ctx, repoURL); err != nil {
		log.Error("анализ завершился с ошибкой", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
