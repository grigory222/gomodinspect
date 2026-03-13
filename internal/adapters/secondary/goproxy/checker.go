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

// VersionChecker отвечает за получение последней версии модуля
type VersionChecker struct {
	proxyURL   string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewVersionChecker(logger *slog.Logger) *VersionChecker {
	return &VersionChecker{
		proxyURL:   defaultProxyURL,
		httpClient: http.DefaultClient,
		logger:     logger,
	}
}

// latestInfo описывает ответ Go module proxy.
type latestInfo struct {
	Version string `json:"Version"`
}

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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			v.logger.Warn("не удалось закрыть тело ответа", slog.String("error", err.Error()))
		}
	}()

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
