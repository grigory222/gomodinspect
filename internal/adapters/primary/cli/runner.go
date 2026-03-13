package cli

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/grigory/gomodinspect/internal/ports"
)

// Runner управляет запуском анализа из командной строки
type Runner struct {
	inspector ports.Inspector
	logger    *slog.Logger
}

// NewRunner создаёт новый экземпляр CLI-раннера
func NewRunner(inspector ports.Inspector, logger *slog.Logger) *Runner {
	return &Runner{inspector: inspector, logger: logger}
}

// Run выполняет анализ указанного репозитория и выводит результат в консоль
func (r *Runner) Run(ctx context.Context, repoURL string) error {
	info, err := r.inspector.Inspect(ctx, repoURL)
	if err != nil {
		return fmt.Errorf("анализ: %w", err)
	}

	fmt.Println()
	fmt.Printf("Модуль: %s\n", info.Name)
	fmt.Printf("Версия Go: %s\n", info.GoVersion)
	fmt.Printf("Данные актуальны на: %s\n", info.AnalyzedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()

	var hasUpdates bool
	for _, d := range info.Deps {
		if d.UpdateAvail {
			hasUpdates = true
			break
		}
	}

	if !hasUpdates {
		fmt.Println("Все зависимости актуальны!")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "МОДУЛЬ\tТЕКУЩАЯ\tПОСЛЕДНЯЯ"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "------\t--------\t---------"); err != nil {
		return err
	}
	for _, d := range info.Deps {
		if d.UpdateAvail {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", d.Name, d.CurrentVersion, d.LatestVersion); err != nil {
				return err
			}
		}
	}
	return w.Flush()
}

// RunInteractive запускает интерактивный режим
func (r *Runner) RunInteractive(ctx context.Context) {
	fmt.Println("Интерактивный режим. Введите URL GitHub-репозитория. \\q — выход, \\h — помощь.")

	lines := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	for {
		fmt.Print("> ")
		select {
		case <-ctx.Done():
			fmt.Println("\nВыход.")
			return
		case line, ok := <-lines:
			if !ok {
				return
			}
			line = strings.TrimSpace(line)
			switch line {
			case `\q`, "":
				return
			case `\h`:
				fmt.Println("Введите URL репозитория, например: https://github.com/gin-gonic/gin")
				fmt.Println(`\q — выход`)
			default:
				if err := r.Run(ctx, line); err != nil {
					r.logger.Error("ошибка анализа", slog.String("error", err.Error()))
				}
			}
		}
	}
}
