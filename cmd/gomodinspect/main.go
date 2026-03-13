package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
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
	"github.com/grigory/gomodinspect/internal/ports"
)

func main() {
	interactive := flag.Bool("interactive", false, "интерактивный режим")
	fastMode := flag.Bool("fast-mode", false, "использовать кеш (быстрый режим)")
	flag.Parse()

	if !*interactive && flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	_ = godotenv.Load()

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg := config.MustLoad(cfgPath)
	log := logger.New(cfg.Log.Level)

	ctx, cancel := context.WithCancel(context.Background())

	var ghClient *gh.Client
	if cfg.GitHub.Token != "" {
		ghClient = gh.NewClient(nil).WithAuthToken(cfg.GitHub.Token)
	} else {
		ghClient = gh.NewClient(nil)
	}

	repoFetcher := ghAdapter.NewRepoFetcher(ghClient, log)
	versionChecker := goproxy.NewVersionChecker(log)

	var cache ports.AnalysisCache
	if *fastMode {
		rdb, err := redisAdapter.Connect(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, log)
		if err != nil {
			log.Error("не удалось подключиться к Redis", slog.String("error", err.Error()))
			os.Exit(1)
		}
		defer func() { _ = rdb.Close() }()
		cache = redisAdapter.NewCache(rdb, cfg.Redis.TTL, log)
	}

	inspector := services.NewInspector(repoFetcher, versionChecker, cache, cfg.Inspector.Workers, log)
	runner := cli.NewRunner(inspector, log)

	var wg sync.WaitGroup

	if *interactive && *fastMode {
		wg.Add(1)
		go func() {
			defer wg.Done()
			services.NewRefresher(inspector, cache, cfg.Redis.RefreshInterval, log).Start(ctx)
		}()
	}

	if *interactive {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runner.RunInteractive(ctx)
			cancel()
		}()
	} else {
		if err := runner.Run(ctx, flag.Arg(0)); err != nil {
			log.Error("анализ завершился с ошибкой", slog.String("error", err.Error()))
		}
		return
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info("получен сигнал завершения", slog.String("signal", sig.String()))
	case <-ctx.Done():
	}

	cancel()
	wg.Wait()
}
