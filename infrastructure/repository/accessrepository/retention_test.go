package accessrepository_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// retentionCase names one stale-session delete outcome against mocked PostgreSQL.
type retentionCase struct {
	name        string
	limit       int
	expectSQL   bool
	execResult  error
	affected    int64
	wantRemoved int64
	wantErr     bool
}

// TestDeleteStaleUsesBoundedPredicate shows the strict cutoff, the row limit and the single-table scope in SQL.
func TestDeleteStaleUsesBoundedPredicate(t *testing.T) {
	cutoff := testdata.Now()
	tests := []retentionCase{
		{name: "batch removed", limit: 500, expectSQL: true, affected: 3, wantRemoved: 3},
		{name: "nothing stale", limit: 500, expectSQL: true, affected: 0, wantRemoved: 0},
		{name: "storage failure is returned", limit: 500, expectSQL: true, execResult: assert.AnError, wantErr: true},
		{name: "non positive limit is rejected before SQL", limit: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := sqlStore(t)
			if tt.expectSQL {
				expected := mock.ExpectExec(`(?i)^DELETE FROM ["']?administrative_sessions["']? WHERE id IN \(SELECT id FROM ["']?administrative_sessions["']? WHERE expires_at < \$1 OR revoked_at < \$2 ORDER BY id LIMIT \$3 FOR UPDATE SKIP LOCKED\)$`).WithArgs(cutoff, cutoff, tt.limit)
				if tt.execResult != nil {
					expected.WillReturnError(tt.execResult)
				} else {
					expected.WillReturnResult(sqlmock.NewResult(0, tt.affected))
				}
			}
			removed, err := store.DeleteStale(context.Background(), cutoff, tt.limit)
			assert.Equal(t, tt.wantRemoved, removed)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
