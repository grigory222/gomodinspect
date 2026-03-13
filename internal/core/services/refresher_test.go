package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/grigory/gomodinspect/internal/core/domain"
	"github.com/grigory/gomodinspect/internal/logger"
	"github.com/grigory/gomodinspect/internal/mocks"
)

func TestRefresher_Start_StopsOnCancel(t *testing.T) {
	log := logger.NewDiscard()
	inspector := mocks.NewMockInspector(t)
	cache := mocks.NewMockAnalysisCache(t)

	r := NewRefresher(inspector, cache, 1*time.Hour, log)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		r.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestRefresher_refresh_Success(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	inspector := mocks.NewMockInspector(t)
	cache := mocks.NewMockAnalysisCache(t)

	cache.EXPECT().AllRepoURLs(mock.Anything).
		Return([]string{"https://github.com/a/b", "https://github.com/c/d"}, nil)

	cache.EXPECT().Delete(mock.Anything, "https://github.com/a/b").Return(nil)
	cache.EXPECT().Delete(mock.Anything, "https://github.com/c/d").Return(nil)

	inspector.EXPECT().Inspect(mock.Anything, "https://github.com/a/b").
		Return(&domain.ModuleInfo{Name: "a/b"}, nil)
	inspector.EXPECT().Inspect(mock.Anything, "https://github.com/c/d").
		Return(&domain.ModuleInfo{Name: "c/d"}, nil)

	r := NewRefresher(inspector, cache, 1*time.Hour, log)
	r.refresh(ctx)
}

func TestRefresher_refresh_AllRepoURLsError(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	inspector := mocks.NewMockInspector(t)
	cache := mocks.NewMockAnalysisCache(t)

	cache.EXPECT().AllRepoURLs(mock.Anything).
		Return(nil, errors.New("redis down"))

	r := NewRefresher(inspector, cache, 1*time.Hour, log)
	r.refresh(ctx)
}

func TestRefresher_refresh_DeleteError(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	inspector := mocks.NewMockInspector(t)
	cache := mocks.NewMockAnalysisCache(t)

	cache.EXPECT().AllRepoURLs(mock.Anything).
		Return([]string{"https://github.com/a/b", "https://github.com/c/d"}, nil)

	cache.EXPECT().Delete(mock.Anything, "https://github.com/a/b").
		Return(errors.New("delete failed"))
	cache.EXPECT().Delete(mock.Anything, "https://github.com/c/d").
		Return(nil)

	inspector.EXPECT().Inspect(mock.Anything, "https://github.com/c/d").
		Return(&domain.ModuleInfo{Name: "c/d"}, nil)

	r := NewRefresher(inspector, cache, 1*time.Hour, log)
	r.refresh(ctx)
}

func TestRefresher_refresh_InspectError(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	inspector := mocks.NewMockInspector(t)
	cache := mocks.NewMockAnalysisCache(t)

	cache.EXPECT().AllRepoURLs(mock.Anything).
		Return([]string{"https://github.com/a/b"}, nil)

	cache.EXPECT().Delete(mock.Anything, "https://github.com/a/b").Return(nil)

	inspector.EXPECT().Inspect(mock.Anything, "https://github.com/a/b").
		Return(nil, errors.New("inspect failed"))

	r := NewRefresher(inspector, cache, 1*time.Hour, log)
	r.refresh(ctx)
}

func TestRefresher_refresh_EmptyURLs(t *testing.T) {
	ctx := context.Background()
	log := logger.NewDiscard()

	inspector := mocks.NewMockInspector(t)
	cache := mocks.NewMockAnalysisCache(t)

	cache.EXPECT().AllRepoURLs(mock.Anything).
		Return([]string{}, nil)

	r := NewRefresher(inspector, cache, 1*time.Hour, log)
	r.refresh(ctx)
}
