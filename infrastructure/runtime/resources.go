package runtime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/taskrepository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type connectionCleanup struct {
	done   <-chan struct{}
	socket net.Conn
}
type databaseResources struct {
	pool        *sql.DB
	mu          sync.Mutex
	connections []connectionCleanup
	closing     bool
}

func (r *databaseResources) connected(_ context.Context, connection *pgx.Conn) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing {
		return errors.New("database is closing")
	}
	// Keep only outstanding cleanup so connection churn does not grow the tracker.
	active := r.connections[:0]
	for _, previous := range r.connections {
		select {
		case <-previous.done:
		default:
			active = append(active, previous)
		}
	}
	r.connections = append(active, connectionCleanup{done: connection.PgConn().CleanupDone(), socket: connection.PgConn().Conn()})
	return nil
}
func (r *databaseResources) close(ctx context.Context) error {
	var result error
	r.mu.Lock()
	r.closing = true
	r.mu.Unlock()
	if err := r.pool.Close(); err != nil {
		result = fmt.Errorf("close database: %w", err)
	}
	r.mu.Lock()
	connections := append([]connectionCleanup(nil), r.connections...)
	r.mu.Unlock()
	for _, connection := range connections {
		select {
		case <-connection.done:
			continue
		case <-ctx.Done():
			// A forced socket close cannot establish driver goroutine completion; retain
			// deadline failure, and let the executable watchdog enforce process policy.
			if err := connection.socket.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				result = errors.Join(result, err)
			}
			result = errors.Join(result, fmt.Errorf("database driver cleanup incomplete: %w", ctx.Err()))
		}
	}
	return result
}
func openDatabase(ctx context.Context, url string, beginCleanup func()) (_ *gorm.DB, resources *databaseResources, result error) {
	config, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, nil, fmt.Errorf("database configuration: %w", err)
	}
	resources = &databaseResources{}
	resources.pool = stdlib.OpenDB(*config, stdlib.OptionAfterConnect(resources.connected))
	defer func() {
		if result != nil {
			beginCleanup()
			cleanupCtx, cancel := context.WithTimeout(context.Background(), ShutdownBudget)
			defer cancel()
			result = errors.Join(result, resources.close(cleanupCtx))
		}
	}()
	resources.pool.SetMaxOpenConns(10)
	resources.pool.SetMaxIdleConns(5)
	resources.pool.SetConnMaxLifetime(30 * time.Minute)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: resources.pool}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), DisableAutomaticPing: true})
	if err != nil {
		return nil, resources, fmt.Errorf("open database: %w", err)
	}
	if err = resources.pool.PingContext(ctx); err != nil {
		return nil, resources, fmt.Errorf("database ping: %w", err)
	}
	if err = db.WithContext(ctx).AutoMigrate(&taskrepository.Model{}); err != nil {
		return nil, resources, fmt.Errorf("database migration: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return nil, resources, err
	}
	return db, resources, nil
}
