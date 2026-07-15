package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"caatsm/internal/adapter/dto"
	"caatsm/internal/adapter/mapper"
	"caatsm/internal/port"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

// --- fake pool -------------------------------------------------------------

type fakeDB struct {
	dedupErr            error // error returned by the telegram_keys INSERT
	telegramErr        error // error returned by the telegrams INSERT
	dedupInsertCalled  bool
	telegramInsertCalled bool
	directInsertCalled bool
	committed          bool
	rolledBack         bool
}

func (f *fakeDB) Begin(_ context.Context) (pgx.Tx, error) {
	return &fakeTx{db: f}, nil
}

// Exec backs the direct-insert path (business key missing).
func (f *fakeDB) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	f.directInsertCalled = true
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (f *fakeDB) QueryRow(_ context.Context, _ string, _ ...interface{}) pgx.Row {
	return nil
}

func (f *fakeDB) CopyFrom(_ context.Context, _ pgx.Identifier, _ []string, _ pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

type fakeTx struct {
	pgx.Tx // embedded to satisfy pgx.Tx; only the methods below are exercised
	db *fakeDB
}

func (t *fakeTx) Exec(_ context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
	switch {
	case strings.Contains(sql, "telegram_keys"):
		t.db.dedupInsertCalled = true
		if t.db.dedupErr != nil {
			return pgconn.CommandTag{}, t.db.dedupErr
		}
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	case strings.Contains(sql, "telegrams"):
		t.db.telegramInsertCalled = true
		if t.db.telegramErr != nil {
			return pgconn.CommandTag{}, t.db.telegramErr
		}
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (t *fakeTx) QueryRow(_ context.Context, _ string, _ ...interface{}) pgx.Row {
	return nil
}

func (t *fakeTx) Commit(_ context.Context) error {
	t.db.committed = true
	return nil
}

func (t *fakeTx) Rollback(_ context.Context) error {
	if !t.db.committed {
		t.db.rolledBack = true
	}
	return nil
}

// --- helpers ---------------------------------------------------------------

func newTestRepo(db dbExecer) *Repository {
	return &Repository{pool: db, mapper: mapper.NewTelegramMapper(), logger: zap.NewNop()}
}

func sampleMsg() *dto.ParsedTelegram {
	return &dto.ParsedTelegram{
		Uuid:       uuid.NewString(),
		MessageID:  "TMQ1234",
		DateTime:   "150631",
		Category:   "ARR",
		Content:    "ZCZC TMQ1234 150631",
		Status:     dto.MessageStatusParsed,
		Parsed:     true,
		ReceivedAt: time.Now(),
		ParsedAt:   time.Now(),
	}
}

// --- characterization tests -------------------------------------------------

// Business key present, gate reserves, telegram inserted, transaction committed.
func TestInsertOne_DedupKeyPresent_InsertsAndCommits(t *testing.T) {
	db := &fakeDB{}
	repo := newTestRepo(db)
	if err := repo.InsertOne(context.Background(), sampleMsg()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !db.dedupInsertCalled {
		t.Error("expected dedup-key reservation to be called")
	}
	if !db.telegramInsertCalled {
		t.Error("expected telegram insert to be called")
	}
	if db.directInsertCalled {
		t.Error("did not expect direct insert on the dedup path")
	}
	if !db.committed {
		t.Error("expected transaction to be committed")
	}
	if db.rolledBack {
		t.Error("did not expect rollback on success")
	}
}

// Concurrent/duplicate writer: gate PK violation => duplicate, telegram skipped,
// ErrDuplicate returned, transaction rolled back.
func TestInsertOne_DedupKeyConflict_ReturnsErrDuplicateSkipsTelegram(t *testing.T) {
	db := &fakeDB{dedupErr: &pgconn.PgError{Code: uniqueViolationCode, Message: "duplicate key"}}
	repo := newTestRepo(db)
	if err := repo.InsertOne(context.Background(), sampleMsg()); err != port.ErrDuplicate {
		t.Fatalf("duplicate must return port.ErrDuplicate, got: %v", err)
	}
	if !db.dedupInsertCalled {
		t.Error("expected dedup-key reservation to be attempted")
	}
	if db.telegramInsertCalled {
		t.Error("telegram must NOT be inserted on duplicate")
	}
	if db.committed {
		t.Error("did not expect commit on duplicate")
	}
	if !db.rolledBack {
		t.Error("expected rollback on duplicate")
	}
}

// Incomplete business identity (missing message_id) falls back to direct insert,
// bypassing the gate, preserving legacy behaviour.
func TestInsertOne_MissingBusinessKey_InsertsDirect(t *testing.T) {
	db := &fakeDB{}
	repo := newTestRepo(db)
	msg := sampleMsg()
	msg.MessageID = ""
	if err := repo.InsertOne(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !db.directInsertCalled {
		t.Error("expected direct insert when business key is incomplete")
	}
	if db.dedupInsertCalled {
		t.Error("did not expect dedup gate on incomplete business key")
	}
}

// Telegram insert failure inside the transaction must roll back (no dangling
// gate row) and surface the error.
func TestInsertOne_TelegramInsertError_RollsBack(t *testing.T) {
	db := &fakeDB{telegramErr: context.DeadlineExceeded}
	repo := newTestRepo(db)
	if err := repo.InsertOne(context.Background(), sampleMsg()); err == nil {
		t.Fatal("expected error from telegram insert failure")
	}
	if !db.dedupInsertCalled {
		t.Error("expected dedup-key reservation before telegram insert")
	}
	if !db.telegramInsertCalled {
		t.Error("expected telegram insert to be attempted")
	}
	if db.committed {
		t.Error("did not expect commit on failure")
	}
	if !db.rolledBack {
		t.Error("expected rollback on failure")
	}
}

// Non-unique error on the gate (e.g. transient DB error) must surface, not be
// silently treated as a duplicate.
func TestInsertOne_DedupInsertError_ReturnsError(t *testing.T) {
	db := &fakeDB{dedupErr: context.Canceled}
	repo := newTestRepo(db)
	if err := repo.InsertOne(context.Background(), sampleMsg()); err == nil {
		t.Fatal("expected error from non-unique dedup failure")
	}
	if db.committed {
		t.Error("did not expect commit on failure")
	}
	if !db.rolledBack {
		t.Error("expected rollback on failure")
	}
}
