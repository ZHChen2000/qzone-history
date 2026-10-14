package usecase

import (
	"context"
	"errors"
	"path/filepath"
	"qzone-history/internal/domain/entity"
	"qzone-history/internal/domain/repository"
	"qzone-history/internal/infrastructure/persistence"
	"qzone-history/internal/infrastructure/qzone_api"
	"qzone-history/pkg/database"
	"qzone-history/pkg/database/sqlite"
	"testing"
	"time"
)

type batchAPI struct {
	qzone_api.QzoneAPIClient
	fetch func(qzone_api.FetchOptions) ([]*entity.Activity, error)
}

func (api batchAPI) GetAllActivities(_ map[string]string, opts qzone_api.FetchOptions) ([]*entity.Activity, error) {
	return api.fetch(opts)
}

type batchRepo struct {
	repository.ActivityRepository
	calls   int
	saved   []entity.Activity
	failure error
}

func (repo *batchRepo) BatchImport(ctx context.Context, batch []entity.Activity) error {
	repo.calls++
	if repo.failure != nil {
		return repo.failure
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	repo.saved = append(repo.saved, batch...)
	return nil
}
func sampleActivity() *entity.Activity {
	return &entity.Activity{SenderQQ: "friend", ReceiverQQ: "self", Content: "sample", TimeText: "2020年1月17日 12:00", Timestamp: time.Date(2020, 1, 17, 12, 0, 0, 0, time.UTC)}
}

func TestActivityCheckpointSurvivesCancellationOnDisk(t *testing.T) {
	file := filepath.Join(t.TempDir(), "app.db")
	db := sqlite.NewSQLiteDB()
	if err := db.Connect(&database.Config{DBName: file}); err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewActivityRepository(db)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api := batchAPI{fetch: func(opts qzone_api.FetchOptions) ([]*entity.Activity, error) {
		batch := []*entity.Activity{sampleActivity()}
		if opts.OnBatch == nil {
			t.Fatal("checkpoint callback absent")
		}
		if err := opts.OnBatch(batch); err != nil {
			t.Fatal(err)
		}
		rows, err := repo.FindByUserQQ(context.Background(), "self", -1, 0)
		if err != nil || len(rows) != 1 {
			t.Fatal("data not saved during scan")
		}
		cancel()
		return batch, context.Canceled
	}}
	uc := NewActivityUseCase(api, repo)
	got, err := uc.FetchActivities(ctx, entity.User{QQ: "self"}, 25000, 2017)
	if !errors.Is(err, context.Canceled) || len(got) != 1 {
		t.Fatalf("canceled result: %d %v", len(got), err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := sqlite.NewSQLiteDB()
	if err := reopened.Connect(&database.Config{DBName: file}); err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	rows, err := persistence.NewActivityRepository(reopened).FindByUserQQ(context.Background(), "self", -1, 0)
	if err != nil || len(rows) != 1 || rows[0].ID == "" {
		t.Fatalf("checkpoint not durable: rows=%d error=%v", len(rows), err)
	}
}

func TestStreamedActivitiesAreNotWrittenTwice(t *testing.T) {
	repo := &batchRepo{}
	api := batchAPI{fetch: func(opts qzone_api.FetchOptions) ([]*entity.Activity, error) {
		batch := []*entity.Activity{sampleActivity()}
		if err := opts.OnBatch(batch); err != nil {
			return batch, err
		}
		return batch, nil
	}}
	result, err := NewActivityUseCase(api, repo).FetchActivities(context.Background(), entity.User{QQ: "self"}, 25000, 2017)
	if err != nil || len(result) != 1 || repo.calls != 1 || len(repo.saved) != 1 {
		t.Fatalf("duplicate writes: %d %d %v", repo.calls, len(repo.saved), err)
	}
}

func TestNonStreamingClientStillPersistsActivities(t *testing.T) {
	repo := &batchRepo{}
	api := batchAPI{fetch: func(qzone_api.FetchOptions) ([]*entity.Activity, error) {
		return []*entity.Activity{sampleActivity()}, nil
	}}
	_, err := NewActivityUseCase(api, repo).FetchActivities(context.Background(), entity.User{QQ: "self"}, 25000, 2017)
	if err != nil || len(repo.saved) != 1 {
		t.Fatalf("nonstreaming compatibility: %v", err)
	}
}

func TestActivityCheckpointFailureIsReported(t *testing.T) {
	expected := errors.New("disk error")
	repo := &batchRepo{failure: expected}
	api := batchAPI{fetch: func(opts qzone_api.FetchOptions) ([]*entity.Activity, error) {
		batch := []*entity.Activity{sampleActivity()}
		return batch, opts.OnBatch(batch)
	}}
	_, err := NewActivityUseCase(api, repo).FetchActivities(context.Background(), entity.User{QQ: "self"}, 25000, 2017)
	if !errors.Is(err, expected) || repo.calls != 1 {
		t.Fatalf("write failure hidden or retried: calls=%d error=%v", repo.calls, err)
	}
}
