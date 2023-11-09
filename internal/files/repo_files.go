package files

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/moov-io/base/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Repository interface {
	Record(ctx context.Context, file AcceptedFile) error
	Cancel(ctx context.Context, fileID string) error
}

func NewRepository(db *sql.DB) Repository {
	if db == nil {
		return &MockRepository{}
	}
	return &sqlRepository{db: db}
}

type sqlRepository struct {
	db *sql.DB
}

func (r *sqlRepository) Record(ctx context.Context, file AcceptedFile) error {
	ctx, span := telemetry.StartSpan(ctx, "files-record", trace.WithAttributes(
		attribute.String("achgateway.file_id", file.FileID),
	))
	defer span.End()

	qry := `INSERT INTO files (file_id, shard_key, hostname, accepted_at) VALUES (?,?,?,?);`
	_, err := r.db.ExecContext(ctx, qry,
		file.FileID,
		file.ShardKey,
		file.Hostname,
		file.AcceptedAt,
	)
	if err != nil {
		return fmt.Errorf("recording file failed: %w", err)
	}
	return nil
}

func (r *sqlRepository) Cancel(ctx context.Context, fileID string) error {
	ctx, span := telemetry.StartSpan(ctx, "files-cancel", trace.WithAttributes(
		attribute.String("achgateway.file_id", fileID),
	))
	defer span.End()

	qry := `UPDATE files SET canceled_at = ? WHERE file_id = ? AND canceled_at IS NULL;`
	_, err := r.db.ExecContext(ctx, qry,
		// SET
		time.Now().In(time.UTC),
		// WHERE
		fileID,
	)
	if err != nil {
		return fmt.Errorf("saving file cancellation failed: %w", err)
	}
	return nil
}
