package files

import (
	"context"
	"testing"
	"time"

	"github.com/moov-io/achgateway/internal/dbtest"
	"github.com/moov-io/base"
	"github.com/moov-io/base/database"

	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("-short flag was specified")
	}

	conf := dbtest.CreateTestDatabase(t, dbtest.LocalDatabaseConfig())
	db := dbtest.LoadDatabase(t, conf)
	require.NoError(t, db.Ping())

	repo := NewRepository(db)
	if _, ok := repo.(*sqlRepository); !ok {
		t.Errorf("unexpected repository type: %T", repo)
	}

	ctx := context.Background()
	fileID1 := base.ID()
	accepted := AcceptedFile{
		FileID:     fileID1,
		ShardKey:   base.ID(),
		Hostname:   "achgateway-0",
		AcceptedAt: time.Now(),
	}

	// Record
	err := repo.Record(ctx, accepted)
	require.NoError(t, err)

	err = repo.Record(ctx, accepted)
	require.ErrorContains(t, err, "Duplicate entry")
	require.True(t, database.UniqueViolation(err))

	// Second File
	fileID2 := base.ID()
	accepted.FileID = fileID2
	err = repo.Record(ctx, accepted)
	require.NoError(t, err)

	// Cancel
	err = repo.Cancel(ctx, fileID1)
	require.NoError(t, err)

	err = repo.Cancel(ctx, base.ID())
	require.NoError(t, err)
}
