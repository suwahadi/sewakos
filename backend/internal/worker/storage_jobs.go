package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/platform/storage"
)

const (
	uploadsPrefix           = "uploads/"
	photosPrefix            = "properties/"
	staleObjectAge          = 24 * time.Hour
	maxPurgeBackoff         = 24 * time.Hour
	purgeAttemptsBeforeLog  = 10
	cleanupUploadsLockKey   = int64(7301)
	reconcileStorageLockKey = int64(7302)
)

type ObjectStore interface {
	Remove(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]storage.Object, error)
}

func inTx(ctx context.Context, deps Deps, fn func(*dbgen.Queries) error) error {
	tx, err := deps.Booking.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(dbgen.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func purgeBackoff(attempts int16) time.Duration {
	exponent := min(int(attempts), 11)
	return min(time.Duration(1<<exponent)*time.Minute, maxPurgeBackoff)
}

func PurgeStorage(ctx context.Context, deps Deps, logger *slog.Logger) {
	var rows []dbgen.SelectDueStorageDeletionsRow
	err := inTx(ctx, deps, func(q *dbgen.Queries) (err error) {
		rows, err = q.SelectDueStorageDeletions(ctx)
		return err
	})
	if err != nil {
		logger.Error("purge_storage: select", "error", err)
		return
	}
	for _, row := range rows {
		if err := purgeOne(ctx, deps, logger, row); err != nil {
			logger.Error("purge_storage: process", "storage_deletion_id", row.ID, "error", err)
		}
	}
}

func purgeOne(ctx context.Context, deps Deps, logger *slog.Logger, row dbgen.SelectDueStorageDeletionsRow) error {
	removeErr := deps.Storage.Remove(ctx, row.ObjectKey)
	if removeErr == nil {
		return inTx(ctx, deps, func(q *dbgen.Queries) error { return q.DeleteStorageDeletion(ctx, row.ID) })
	}
	attempts := row.Attempts + 1
	if attempts >= purgeAttemptsBeforeLog {
		logger.Error("purge_storage: object still not deleted", "object_key", row.ObjectKey, "attempts", attempts, "error", removeErr)
	}
	return inTx(ctx, deps, func(q *dbgen.Queries) error {
		return q.MarkStorageDeletionFailed(ctx, dbgen.MarkStorageDeletionFailedParams{
			ID:            row.ID,
			NextAttemptAt: pgtype.Timestamptz{Time: time.Now().Add(purgeBackoff(attempts)), Valid: true},
			LastError:     pgtype.Text{String: removeErr.Error(), Valid: true},
		})
	})
}

func CleanupUploads(ctx context.Context, deps Deps, logger *slog.Logger, now time.Time) {
	objects, err := deps.Storage.List(ctx, uploadsPrefix)
	if err != nil {
		logger.Error("cleanup_uploads: list", "error", err)
		return
	}
	for _, object := range objects {
		if now.Sub(object.LastModified) <= staleObjectAge {
			continue
		}
		if err := deps.Storage.Remove(ctx, object.Key); err != nil {
			logger.Error("cleanup_uploads: remove", "object_key", object.Key, "error", err)
		}
	}
}

func ReconcileStorage(ctx context.Context, deps Deps, logger *slog.Logger, now time.Time) {
	var photos []dbgen.ListPhotoObjectKeysRow
	if err := inTx(ctx, deps, func(q *dbgen.Queries) (err error) {
		photos, err = q.ListPhotoObjectKeys(ctx)
		return err
	}); err != nil {
		logger.Error("reconcile_storage: list photos", "error", err)
		return
	}
	objects, err := deps.Storage.List(ctx, photosPrefix)
	if err != nil {
		logger.Error("reconcile_storage: list objects", "error", err)
		return
	}

	referenced := make(map[string]bool, 2*len(photos))
	for _, photo := range photos {
		referenced[photo.ObjectKey], referenced[photo.ThumbKey] = true, true
	}
	stored := make(map[string]bool, len(objects))
	var orphans []string
	for _, object := range objects {
		stored[object.Key] = true
		if !referenced[object.Key] && now.Sub(object.LastModified) > staleObjectAge {
			orphans = append(orphans, object.Key)
		}
	}
	if err := inTx(ctx, deps, func(q *dbgen.Queries) error {
		for _, key := range orphans {
			if err := q.QueueStorageDeletion(ctx, key); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		logger.Error("reconcile_storage: queue orphans", "error", err)
	}
	for _, photo := range photos {
		for _, key := range []string{photo.ObjectKey, photo.ThumbKey} {
			if !stored[key] {
				logger.Error("reconcile_storage: photo object missing", "photo_id", photo.ID, "object_key", key)
			}
		}
	}
}

type Scheduler struct {
	dailyDoneOn  string
	weeklyDoneOn string
}

func (s *Scheduler) RunDue(ctx context.Context, deps Deps, logger *slog.Logger, now time.Time) {
	wib := now.In(money.WIB)
	today := wib.Format(time.DateOnly)
	if s.dailyDoneOn != today && wib.Hour() >= 2 {
		s.dailyDoneOn = today
		runExclusive(ctx, deps, logger, cleanupUploadsLockKey, func() { CleanupUploads(ctx, deps, logger, now) })
	}
	if s.weeklyDoneOn != today && wib.Weekday() == time.Sunday && wib.Hour() >= 3 {
		s.weeklyDoneOn = today
		runExclusive(ctx, deps, logger, reconcileStorageLockKey, func() { ReconcileStorage(ctx, deps, logger, now) })
	}
}

// The transaction stays open for the whole job so its advisory lock keeps other worker instances out until it ends.
func runExclusive(ctx context.Context, deps Deps, logger *slog.Logger, lockKey int64, job func()) {
	tx, err := deps.Booking.DB.Begin(ctx)
	if err != nil {
		logger.Error("scheduled job: begin", "lock_key", lockKey, "error", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	locked, err := dbgen.New(tx).TryAdvisoryXactLock(ctx, lockKey)
	if err != nil {
		logger.Error("scheduled job: advisory lock", "lock_key", lockKey, "error", err)
		return
	}
	if locked {
		job()
	}
}
