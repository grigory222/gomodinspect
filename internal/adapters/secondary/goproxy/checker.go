// Package goproxy — вторичный адаптер (secondary) для проверки версий через Go module proxy.
package goproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

const defaultProxyURL = "https://proxy.golang.org"

// VersionChecker реализует порт ports.VersionChecker через Go module proxy.
type VersionChecker struct {
	proxyURL   string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewVersionChecker создаёт новый VersionChecker с прокси по умолчанию.
func NewVersionChecker(logger *slog.Logger) *VersionChecker {
	return &VersionChecker{
		proxyURL:   defaultProxyURL,
		httpClient: http.DefaultClient,
		logger:     logger,
	}
}

// NewVersionCheckerWithURL создаёт VersionChecker с указанным URL прокси (для тестирования).
func NewVersionCheckerWithURL(proxyURL string, httpClient *http.Client, logger *slog.Logger) *VersionChecker {
	return &VersionChecker{
		proxyURL:   proxyURL,
		httpClient: httpClient,
		logger:     logger,
	}
}

// latestInfo описывает ответ Go module proxy.
type latestInfo struct {
	Version string `json:"Version"`
}

// GetLatestVersion запрашивает последнюю версию модуля у Go module proxy.
func (v *VersionChecker) GetLatestVersion(ctx context.Context, modulePath string) (string, error) {
	url := fmt.Sprintf("%s/%s/@latest", v.proxyURL, modulePath)

	v.logger.Debug("запрос к go proxy", slog.String("url", url))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("создание запроса: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("запрос к прокси: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("прокси вернул %d: %s", resp.StatusCode, string(body))
	}

	var info latestInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("декодирование ответа прокси: %w", err)
	}

	return info.Version, nil
}
