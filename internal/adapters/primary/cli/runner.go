// Package cli — первичный адаптер (primary), реализующий интерфейс командной строки.
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"

	"github.com/grigory/gomodinspect/internal/ports"
)

// Runner управляет запуском анализа из командной строки.
type Runner struct {
	inspector ports.Inspector
	logger    *slog.Logger
}

// NewRunner создаёт новый экземпляр CLI-раннера.
func NewRunner(inspector ports.Inspector, logger *slog.Logger) *Runner {
	return &Runner{inspector: inspector, logger: logger}
}

// Run выполняет анализ указанного репозитория и выводит результат в консоль.
func (r *Runner) Run(ctx context.Context, repoURL string) error {
	info, err := r.inspector.Inspect(ctx, repoURL)
	if err != nil {
		return fmt.Errorf("анализ: %w", err)
	}

	fmt.Println()
	fmt.Printf("Модуль: %s\n", info.Name)
	fmt.Printf("Версия Go: %s\n", info.GoVersion)
	fmt.Println()

	// Считаем зависимости с доступными обновлениями
	var updatable int
	for _, d := range info.Deps {
		if d.UpdateAvail {
			updatable++
		}
	}

	if updatable == 0 {
		fmt.Println("Все зависимости актуальны!")
		return nil
	}

	fmt.Printf("Зависимости с доступными обновлениями (%d):\n\n", updatable)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "МОДУЛЬ\tТЕКУЩАЯ\tПОСЛЕДНЯЯ")
	fmt.Fprintln(w, "------\t--------\t---------")
	for _, d := range info.Deps {
		if d.UpdateAvail {
			fmt.Fprintf(w, "%s\t%s\t%s\n", d.Name, d.CurrentVersion, d.LatestVersion)
		}
	}
	w.Flush()

	return nil
}
