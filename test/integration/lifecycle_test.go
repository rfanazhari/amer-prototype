package integration_test

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/google/uuid"
	appruntime "github.com/rfanazhari/amer-prototype/infrastructure/runtime"
	"github.com/stretchr/testify/require"
)

func databaseURL(t *testing.T, label string) string {
	t.Helper()
	parsed, err := url.Parse(os.Getenv("TEST_DATABASE_URL"))
	require.NoError(t, err)
	query := parsed.Query()
	query.Set("application_name", label)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
func TestPartialStartupFailureClosesDatabase(t *testing.T) {
	repository := store(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { require.NoError(t, listener.Close()) }()
	label := "amer_test_" + uuid.NewString()
	logs := logmanager.NewApplication(logmanager.WithAppName("lifecycle-test"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = appruntime.Run(ctx, appruntime.Config{DatabaseURL: databaseURL(t, label), Address: listener.Addr().String()}, logs)
	require.ErrorContains(t, err, "listen")
	require.Eventually(t, func() bool {
		var count int64
		err := repository.DB.Raw("SELECT count(*) FROM pg_stat_activity WHERE application_name = ?", label).Scan(&count).Error
		return err == nil && count == 0
	}, time.Second, 10*time.Millisecond)
}
func TestCancellationDuringMigrationClosesAcquisitions(t *testing.T) {
	repository := store(t)
	lock := repository.DB.Begin()
	require.NoError(t, lock.Error)
	defer func() { require.NoError(t, lock.Rollback().Error) }()
	require.NoError(t, lock.Exec("LOCK TABLE tasks IN ACCESS EXCLUSIVE MODE").Error)
	label := "amer_test_" + uuid.NewString()
	logs := logmanager.NewApplication(logmanager.WithAppName("lifecycle-test"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan net.Addr, 1)
	result := make(chan error, 1)
	cfg := appruntime.Config{DatabaseURL: databaseURL(t, label), Address: "127.0.0.1:0", Ready: ready}
	go func() { result <- appruntime.Run(ctx, cfg, logs) }()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	poll := time.NewTicker(10 * time.Millisecond)
	defer poll.Stop()
	waiting := false
	for !waiting {
		select {
		case <-deadline.C:
			t.Fatal("startup did not reach locked migration")
		case <-poll.C:
			var count int64
			require.NoError(t, repository.DB.Raw("SELECT count(*) FROM pg_stat_activity WHERE application_name = ? AND wait_event_type = 'Lock'", label).Scan(&count).Error)
			waiting = count > 0
		}
	}
	cancel()
	select {
	case err := <-result:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("startup cancellation did not terminate")
	}
	require.Empty(t, ready)
	require.Eventually(t, func() bool {
		var count int64
		err := repository.DB.Raw("SELECT count(*) FROM pg_stat_activity WHERE application_name = ?", label).Scan(&count).Error
		return err == nil && count == 0
	}, time.Second, 10*time.Millisecond)
}

func TestShutdownCancelsActiveDatabaseRequest(t *testing.T) {
	repository := store(t)
	task := storedTask(t, repository, false)
	lock := repository.DB.Begin()
	require.NoError(t, lock.Error)
	defer func() { require.NoError(t, lock.Rollback().Error) }()
	var lockedID string
	require.NoError(t, lock.Raw("SELECT id FROM tasks WHERE id = ? FOR UPDATE", task.Snapshot().ID).Scan(&lockedID).Error)
	label := "amer_active_" + uuid.NewString()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logs := logmanager.NewApplication(logmanager.WithAppName("lifecycle-test"))
	ready := make(chan net.Addr, 1)
	result := make(chan error, 1)
	go func() {
		result <- appruntime.Run(ctx, appruntime.Config{DatabaseURL: databaseURL(t, label), Address: "127.0.0.1:0", Ready: ready}, logs)
	}()
	var addr net.Addr
	select {
	case addr = <-ready:
	case err := <-result:
		t.Fatalf("startup: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("startup timed out")
	}
	requestDone := make(chan error, 1)
	go func() {
		client := http.Client{Timeout: 7 * time.Second}
		response, err := client.Post("http://"+addr.String()+"/tasks/"+lockedID+"/start", "application/json", nil)
		if response != nil {
			_ = response.Body.Close()
		}
		requestDone <- err
	}()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	poll := time.NewTicker(10 * time.Millisecond)
	defer poll.Stop()
	waiting := false
	for !waiting {
		select {
		case <-deadline.C:
			t.Fatal("request did not reach database lock")
		case <-poll.C:
			var count int64
			require.NoError(t, repository.DB.Raw("SELECT count(*) FROM pg_stat_activity WHERE application_name = ? AND wait_event_type = 'Lock'", label).Scan(&count).Error)
			waiting = count > 0
		}
	}
	beginning := time.Now()
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Less(t, time.Since(beginning), appruntime.ShutdownBudget)
	case <-time.After(6 * time.Second):
		t.Fatal("runtime did not stop")
	}
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("active request did not terminate")
	}
	require.Eventually(t, func() bool {
		var count int64
		err := repository.DB.Raw("SELECT count(*) FROM pg_stat_activity WHERE application_name = ?", label).Scan(&count).Error
		return err == nil && count == 0
	}, time.Second, 10*time.Millisecond)
}
