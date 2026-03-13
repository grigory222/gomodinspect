package github

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	gh "github.com/google/go-github/v69/github"
)

// RepoFetcher отвечает за получение go.mod из репозитория на GitHub
type RepoFetcher struct {
	client *gh.Client
	logger *slog.Logger
}

func NewRepoFetcher(client *gh.Client, logger *slog.Logger) *RepoFetcher {
	return &RepoFetcher{client: client, logger: logger}
}

// parseOwnerRepo извлекает owner и repo из URL репозитория GitHub
// Поддерживаемые форматы:
//   - https://github.com/owner/repo
//   - github.com/owner/repo
//   - owner/repo
func parseOwnerRepo(repoURL string) (owner, repo string, err error) {
	repoURL = strings.TrimSuffix(repoURL, ".git")
	repoURL = strings.TrimSuffix(repoURL, "/")

	for _, prefix := range []string{"https://", "http://", "ssh://", "git@"} {
		repoURL = strings.TrimPrefix(repoURL, prefix)
	}

	repoURL = strings.TrimPrefix(repoURL, "github.com/")
	repoURL = strings.TrimPrefix(repoURL, "github.com:")

	parts := strings.SplitN(repoURL, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("некорректный URL репозитория: не удалось извлечь owner/repo из %q", repoURL)
	}

	return parts[0], parts[1], nil
}

// FetchGoMod получает содержимое go.mod из дефолтной ветки репозитория
func (f *RepoFetcher) FetchGoMod(ctx context.Context, repoURL string) ([]byte, error) {
	owner, repo, err := parseOwnerRepo(repoURL)
	if err != nil {
		return nil, err
	}

	f.logger.Info("получаем go.mod из GitHub",
		slog.String("owner", owner),
		slog.String("repo", repo),
	)

	fileContent, _, _, err := f.client.Repositories.GetContents(
		ctx, owner, repo, "go.mod",
		&gh.RepositoryContentGetOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("github get contents: %w", err)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return nil, fmt.Errorf("декодирование содержимого go.mod: %w", err)
	}

	return []byte(content), nil
}
