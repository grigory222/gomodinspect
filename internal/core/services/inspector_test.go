package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/grigory/gomodinspect/internal/core/domain"
	"github.com/grigory/gomodinspect/internal/logger"
	"github.com/grigory/gomodinspect/internal/mocks"
)

const testGoMod = `module example.com/mymodule

go 1.21

require (
	golang.org/x/text v0.14.0
	golang.org/x/net v0.20.0
)

require (
	golang.org/x/sys v0.16.0 // indirect
)
`

func TestInspector_Inspect_Success(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	fetcher := mocks.NewMockRepoFetcher(t)
	checker := mocks.NewMockVersionChecker(t)

	fetcher.EXPECT().FetchGoMod(mock.Anything, "https://github.com/example/repo").
		Return([]byte(testGoMod), nil)

	checker.EXPECT().GetLatestVersion(mock.Anything, "golang.org/x/text").
		Return("v0.15.0", nil)
	checker.EXPECT().GetLatestVersion(mock.Anything, "golang.org/x/net").
		Return("v0.20.0", nil)

	svc := NewInspector(fetcher, checker, nil, 2, log)
	info, err := svc.Inspect(ctx, "https://github.com/example/repo")

	require.NoError(t, err)
	assert.Equal(t, "example.com/mymodule", info.Name)
	assert.Equal(t, "1.21", info.GoVersion)
	assert.Len(t, info.Deps, 2)

	depsByName := make(map[string]domain.Dependency)
	for _, d := range info.Deps {
		depsByName[d.Name] = d
	}

	textDep := depsByName["golang.org/x/text"]
	assert.Equal(t, "v0.14.0", textDep.CurrentVersion)
	assert.Equal(t, "v0.15.0", textDep.LatestVersion)
	assert.True(t, textDep.UpdateAvail)

	netDep := depsByName["golang.org/x/net"]
	assert.Equal(t, "v0.20.0", netDep.CurrentVersion)
	assert.Equal(t, "v0.20.0", netDep.LatestVersion)
	assert.False(t, netDep.UpdateAvail)
}

func TestInspector_Inspect_FetchError(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	fetcher := mocks.NewMockRepoFetcher(t)
	checker := mocks.NewMockVersionChecker(t)

	fetcher.EXPECT().FetchGoMod(mock.Anything, "https://github.com/bad/repo").
		Return(nil, errors.New("not found"))

	svc := NewInspector(fetcher, checker, nil, 2, log)
	info, err := svc.Inspect(ctx, "https://github.com/bad/repo")

	assert.Nil(t, info)
	assert.ErrorContains(t, err, "получение go.mod")
}

func TestInspector_Inspect_InvalidGoMod(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	fetcher := mocks.NewMockRepoFetcher(t)
	checker := mocks.NewMockVersionChecker(t)

	fetcher.EXPECT().FetchGoMod(mock.Anything, "https://github.com/bad/gomod").
		Return([]byte("invalid content{{{"), nil)

	svc := NewInspector(fetcher, checker, nil, 2, log)
	info, err := svc.Inspect(ctx, "https://github.com/bad/gomod")

	assert.Nil(t, info)
	assert.ErrorContains(t, err, "парсинг go.mod")
}

func TestInspector_Inspect_VersionCheckError(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	gomod := `module example.com/m

go 1.21

require golang.org/x/text v0.14.0
`
	fetcher := mocks.NewMockRepoFetcher(t)
	checker := mocks.NewMockVersionChecker(t)

	fetcher.EXPECT().FetchGoMod(mock.Anything, "https://github.com/example/repo").
		Return([]byte(gomod), nil)
	checker.EXPECT().GetLatestVersion(mock.Anything, "golang.org/x/text").
		Return("", errors.New("proxy error"))

	svc := NewInspector(fetcher, checker, nil, 1, log)
	info, err := svc.Inspect(ctx, "https://github.com/example/repo")

	require.NoError(t, err)
	require.Len(t, info.Deps, 1)
	assert.Equal(t, "v0.14.0", info.Deps[0].LatestVersion)
	assert.False(t, info.Deps[0].UpdateAvail)
}

func TestInspector_Inspect_CacheHit(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	fetcher := mocks.NewMockRepoFetcher(t)
	checker := mocks.NewMockVersionChecker(t)
	cache := mocks.NewMockAnalysisCache(t)

	deps := []domain.Dependency{
		{Name: "golang.org/x/text", CurrentVersion: "v0.14.0", LatestVersion: "v0.15.0", UpdateAvail: true},
	}
	depsJSON, _ := json.Marshal(deps)
	analyzedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	cache.EXPECT().Get(mock.Anything, "https://github.com/example/repo").
		Return(&domain.AnalysisRecord{
			RepoURL:    "https://github.com/example/repo",
			ModuleName: "example.com/mymodule",
			GoVersion:  "1.21",
			AnalyzedAt: analyzedAt,
			DepsJSON:   string(depsJSON),
		}, nil)

	svc := NewInspector(fetcher, checker, cache, 2, log)
	info, err := svc.Inspect(ctx, "https://github.com/example/repo")

	require.NoError(t, err)
	assert.Equal(t, "example.com/mymodule", info.Name)
	assert.Equal(t, "1.21", info.GoVersion)
	assert.Equal(t, analyzedAt, info.AnalyzedAt)
	assert.Len(t, info.Deps, 1)
	assert.True(t, info.Deps[0].UpdateAvail)
}

func TestInspector_Inspect_CacheMiss(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	fetcher := mocks.NewMockRepoFetcher(t)
	checker := mocks.NewMockVersionChecker(t)
	cache := mocks.NewMockAnalysisCache(t)

	cache.EXPECT().Get(mock.Anything, "https://github.com/example/repo").
		Return(nil, nil)

	fetcher.EXPECT().FetchGoMod(mock.Anything, "https://github.com/example/repo").
		Return([]byte(testGoMod), nil)

	checker.EXPECT().GetLatestVersion(mock.Anything, mock.Anything).
		Return("v1.0.0", nil)

	cache.EXPECT().Save(mock.Anything, mock.Anything).
		Return(nil)

	svc := NewInspector(fetcher, checker, cache, 2, log)
	info, err := svc.Inspect(ctx, "https://github.com/example/repo")

	require.NoError(t, err)
	assert.Equal(t, "example.com/mymodule", info.Name)
	assert.Len(t, info.Deps, 2)
}

func TestInspector_Inspect_CacheGetError(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	fetcher := mocks.NewMockRepoFetcher(t)
	checker := mocks.NewMockVersionChecker(t)
	cache := mocks.NewMockAnalysisCache(t)

	cache.EXPECT().Get(mock.Anything, "https://github.com/example/repo").
		Return(nil, errors.New("redis down"))

	fetcher.EXPECT().FetchGoMod(mock.Anything, "https://github.com/example/repo").
		Return([]byte(testGoMod), nil)

	checker.EXPECT().GetLatestVersion(mock.Anything, mock.Anything).
		Return("v1.0.0", nil)

	cache.EXPECT().Save(mock.Anything, mock.Anything).
		Return(nil)

	svc := NewInspector(fetcher, checker, cache, 2, log)
	info, err := svc.Inspect(ctx, "https://github.com/example/repo")

	require.NoError(t, err)
	assert.Equal(t, "example.com/mymodule", info.Name)
}

func TestInspector_Inspect_CacheSaveError(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	gomod := `module example.com/m

go 1.21

require golang.org/x/text v0.14.0
`
	fetcher := mocks.NewMockRepoFetcher(t)
	checker := mocks.NewMockVersionChecker(t)
	cache := mocks.NewMockAnalysisCache(t)

	cache.EXPECT().Get(mock.Anything, "https://github.com/example/repo").
		Return(nil, nil)
	fetcher.EXPECT().FetchGoMod(mock.Anything, "https://github.com/example/repo").
		Return([]byte(gomod), nil)
	checker.EXPECT().GetLatestVersion(mock.Anything, "golang.org/x/text").
		Return("v0.15.0", nil)
	cache.EXPECT().Save(mock.Anything, mock.Anything).
		Return(errors.New("redis write error"))

	svc := NewInspector(fetcher, checker, cache, 1, log)
	info, err := svc.Inspect(ctx, "https://github.com/example/repo")

	require.NoError(t, err)
	assert.Equal(t, "example.com/m", info.Name)
	assert.Len(t, info.Deps, 1)
}

func TestInspector_Inspect_NoDeps(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	gomod := `module example.com/m

go 1.21
`
	fetcher := mocks.NewMockRepoFetcher(t)
	checker := mocks.NewMockVersionChecker(t)

	fetcher.EXPECT().FetchGoMod(mock.Anything, "https://github.com/example/repo").
		Return([]byte(gomod), nil)

	svc := NewInspector(fetcher, checker, nil, 2, log)
	info, err := svc.Inspect(ctx, "https://github.com/example/repo")

	require.NoError(t, err)
	assert.Equal(t, "example.com/m", info.Name)
	assert.Empty(t, info.Deps)
}

func TestInspector_Inspect_OnlyIndirectDeps(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	gomod := `module example.com/m

go 1.21

require (
	golang.org/x/sys v0.16.0 // indirect
)
`
	fetcher := mocks.NewMockRepoFetcher(t)
	checker := mocks.NewMockVersionChecker(t)

	fetcher.EXPECT().FetchGoMod(mock.Anything, "https://github.com/example/repo").
		Return([]byte(gomod), nil)

	svc := NewInspector(fetcher, checker, nil, 2, log)
	info, err := svc.Inspect(ctx, "https://github.com/example/repo")

	require.NoError(t, err)
	assert.Empty(t, info.Deps)
}
