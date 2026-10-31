package integration_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/taskusecase"
	"github.com/rfanazhari/amer-prototype/domain/aggregate"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/event"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/taskrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/clock"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/eventqueue"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func store(t *testing.T) taskrepository.Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, db.AutoMigrate(&taskrepository.Model{}))
	return taskrepository.Store{DB: db}
}
func storedTask(t *testing.T, repository taskrepository.Store, started bool) *aggregate.Task {
	t.Helper()
	title, err := valueobject.NewTitle("integration task")
	require.NoError(t, err)
	task, err := aggregate.NewTask(uuid.NewString(), title, nil, testdata.Now())
	require.NoError(t, err)
	if started {
		require.NoError(t, task.Start())
	}
	require.NoError(t, repository.Create(context.Background(), task))
	return task
}
func TestConcurrentTransitionsAreSerialized(t *testing.T) {
	repository := store(t)
	task := storedTask(t, repository, false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results := make(chan error, 8)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			_, err := repository.Update(ctx, task.Snapshot().ID, func(loaded *aggregate.Task) error { return loaded.Start() })
			results <- err
		}()
	}
	workers.Wait()
	close(results)
	successful := 0
	for err := range results {
		if err == nil {
			successful++
		} else {
			require.ErrorIs(t, err, aggregate.ErrTransition)
		}
	}
	require.Equal(t, 1, successful)
	loaded, err := repository.Find(ctx, task.Snapshot().ID)
	require.NoError(t, err)
	require.Equal(t, enum.InProgress, loaded.Snapshot().Status)
}
func TestQueueRefusalRollsBackDatabaseAndRepeatedCompletionAddsNoEvent(t *testing.T) {
	repository := store(t)
	task := storedTask(t, repository, true)
	before := task.Snapshot()
	events := make(chan event.TaskCompleted, 2)
	queue := &eventqueue.Queue{Capacity: 1, Consume: func(e event.TaskCompleted) { events <- e }}
	queue.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	t.Cleanup(func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		require.NoError(t, queue.Shutdown(cleanupCtx))
	})
	reserved, err := queue.Reserve()
	require.NoError(t, err)
	defer reserved.Cancel()
	operation := taskusecase.Complete{Repository: repository, Clock: clock.System{}, Queue: queue}
	_, err = operation.Execute(ctx, before.ID)
	require.ErrorIs(t, err, service.ErrEventUnavailable)
	loaded, err := repository.Find(ctx, before.ID)
	require.NoError(t, err)
	require.Equal(t, before, loaded.Snapshot())
	reserved.Cancel()
	result, err := operation.Execute(ctx, before.ID)
	require.NoError(t, err)
	require.Equal(t, "done", result.Status)
	_, err = operation.Execute(ctx, before.ID)
	require.ErrorIs(t, err, aggregate.ErrTransition)
	require.NoError(t, queue.Shutdown(ctx))
	require.Len(t, events, 1)
}
func TestCorruptStoredTitleIsTechnicalFailure(t *testing.T) {
	repository := store(t)
	task := storedTask(t, repository, false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, repository.DB.WithContext(ctx).Model(&taskrepository.Model{}).Where("id = ?", task.Snapshot().ID).Update("title", "").Error)
	_, err := repository.Find(ctx, task.Snapshot().ID)
	var technical errs.Technical
	require.True(t, errors.As(err, &technical))
	require.ErrorIs(t, err, valueobject.ErrInvalidTitle)
}

func TestCommitFailureCancelsEventReservation(t *testing.T) {
	repository := store(t)
	task := storedTask(t, repository, true)
	before := task.Snapshot()
	// Names and predicate values are generated UUIDs, not externally supplied SQL.
	name := "verification_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	functionSQL := "CREATE FUNCTION " + name + "() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'verification commit rejected'; END; $$"
	require.NoError(t, repository.DB.Exec(functionSQL).Error)
	t.Cleanup(func() { require.NoError(t, repository.DB.Exec("DROP FUNCTION IF EXISTS "+name+"() CASCADE").Error) })
	triggerSQL := "CREATE CONSTRAINT TRIGGER " + name + " AFTER UPDATE ON tasks DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.id = '" + before.ID + "'::uuid AND NEW.status = 'done') EXECUTE FUNCTION " + name + "()"
	require.NoError(t, repository.DB.Exec(triggerSQL).Error)
	events := make(chan event.TaskCompleted, 1)
	queue := &eventqueue.Queue{Capacity: 1, Consume: func(e event.TaskCompleted) { events <- e }}
	queue.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := (taskusecase.Complete{Repository: repository, Clock: clock.System{}, Queue: queue}).Execute(ctx, before.ID)
	require.Error(t, err)
	loaded, err := repository.Find(ctx, before.ID)
	require.NoError(t, err)
	require.Equal(t, before, loaded.Snapshot())
	reservation, err := queue.Reserve()
	require.NoError(t, err)
	reservation.Cancel()
	require.NoError(t, queue.Shutdown(ctx))
	require.Empty(t, events)
}
