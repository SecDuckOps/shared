// Package sessiontest provides contract-test helpers for session repository
// implementations. Any adapter (SQLite, Postgres, in-memory, …) can call these
// functions to prove it satisfies the repository interfaces defined in the
// session package.
package sessiontest

import (
	"context"
	"testing"

	"github.com/SecDuckOps/shared/core/session"
	"github.com/SecDuckOps/shared/types"
)

// ---------------------------------------------------------------------------
// SessionRepository contract
// ---------------------------------------------------------------------------

// RunSessionRepositoryContract exercises every method of SessionRepository
// and verifies the expected behaviours and error conditions.
func RunSessionRepositoryContract(t *testing.T, repo session.SessionRepository) {
	t.Helper()
	ctx := context.Background()

	t.Run("Create and GetByID round-trip", func(t *testing.T) {
		s := mustNewSession(t)
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create: %v", err)
		}

		got, err := repo.GetByID(ctx, s.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ID != s.ID {
			t.Fatalf("expected ID %s, got %s", s.ID, got.ID)
		}
		if got.Name != s.Name {
			t.Fatalf("expected Name %q, got %q", s.Name, got.Name)
		}
	})

	t.Run("GetByID returns ErrNotFound for missing ID", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "nonexistent-id")
		assertErrCode(t, err, types.ErrCodeNotFound)
	})

	t.Run("Create returns ErrAlreadyExists on duplicate", func(t *testing.T) {
		s := mustNewSession(t)
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("first Create: %v", err)
		}
		err := repo.Create(ctx, s)
		assertErrCode(t, err, types.ErrCodeAlreadyExists)
	})

	t.Run("List returns sessions for a user with pagination", func(t *testing.T) {
		s1 := mustNewSession(t)
		s2 := mustNewSession(t)
		_ = repo.Create(ctx, s1)
		_ = repo.Create(ctx, s2)

		list, err := repo.List(ctx, s1.UserID, 0, 10)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) < 2 {
			t.Fatalf("expected at least 2 sessions, got %d", len(list))
		}
	})

	t.Run("Update persists changes", func(t *testing.T) {
		s := mustNewSession(t)
		_ = repo.Create(ctx, s)

		_ = s.SetMetadata("env", "production")
		s.Version++
		if err := repo.Update(ctx, s); err != nil {
			t.Fatalf("Update: %v", err)
		}

		got, _ := repo.GetByID(ctx, s.ID)
		if got.Metadata["env"] != "production" {
			t.Fatal("expected metadata to persist after Update")
		}
	})

	t.Run("SoftDelete marks session as deleted", func(t *testing.T) {
		s := mustNewSession(t)
		_ = repo.Create(ctx, s)

		if err := repo.SoftDelete(ctx, s.ID); err != nil {
			t.Fatalf("SoftDelete: %v", err)
		}

		got, err := repo.GetByID(ctx, s.ID)
		if err != nil {
			t.Fatalf("GetByID after SoftDelete: %v", err)
		}
		if got.Status != session.SessionStatusDeleted {
			t.Fatalf("expected status %s, got %s", session.SessionStatusDeleted, got.Status)
		}
	})

	t.Run("SoftDelete returns ErrNotFound for missing ID", func(t *testing.T) {
		err := repo.SoftDelete(ctx, "missing-id")
		assertErrCode(t, err, types.ErrCodeNotFound)
	})
}

// ---------------------------------------------------------------------------
// MessageRepository contract
// ---------------------------------------------------------------------------

// RunMessageRepositoryContract exercises every method of MessageRepository.
func RunMessageRepositoryContract(t *testing.T, repo session.MessageRepository) {
	t.Helper()
	ctx := context.Background()

	t.Run("Create and ListBySessionID round-trip", func(t *testing.T) {
		m := mustNewMessage(t, "sess-contract-1")
		if err := repo.Create(ctx, m); err != nil {
			t.Fatalf("Create: %v", err)
		}

		msgs, err := repo.ListBySessionID(ctx, m.SessionID, 0, 10)
		if err != nil {
			t.Fatalf("ListBySessionID: %v", err)
		}
		if len(msgs) < 1 {
			t.Fatal("expected at least 1 message")
		}
	})

	t.Run("CountBySessionID returns correct count", func(t *testing.T) {
		sid := "sess-contract-count"
		m1 := mustNewMessage(t, sid)
		m2 := mustNewMessage(t, sid)
		_ = repo.Create(ctx, m1)
		_ = repo.Create(ctx, m2)

		count, err := repo.CountBySessionID(ctx, sid)
		if err != nil {
			t.Fatalf("CountBySessionID: %v", err)
		}
		if count < 2 {
			t.Fatalf("expected count >= 2, got %d", count)
		}
	})
}

// ---------------------------------------------------------------------------
// CheckpointRepository contract
// ---------------------------------------------------------------------------

// RunCheckpointRepositoryContract exercises every method of CheckpointRepository.
func RunCheckpointRepositoryContract(t *testing.T, repo session.CheckpointRepository) {
	t.Helper()
	ctx := context.Background()

	t.Run("Create and ListBySessionID round-trip", func(t *testing.T) {
		cp := mustNewCheckpoint(t, "sess-cp-1", 0)
		if err := repo.Create(ctx, cp); err != nil {
			t.Fatalf("Create: %v", err)
		}

		cps, err := repo.ListBySessionID(ctx, cp.SessionID)
		if err != nil {
			t.Fatalf("ListBySessionID: %v", err)
		}
		if len(cps) < 1 {
			t.Fatal("expected at least 1 checkpoint")
		}
	})

	t.Run("GetLatestBySessionID returns most recent", func(t *testing.T) {
		sid := "sess-cp-latest"
		cp1 := mustNewCheckpoint(t, sid, 5)
		cp2 := mustNewCheckpoint(t, sid, 10)
		_ = repo.Create(ctx, cp1)
		_ = repo.Create(ctx, cp2)

		latest, err := repo.GetLatestBySessionID(ctx, sid)
		if err != nil {
			t.Fatalf("GetLatestBySessionID: %v", err)
		}
		if latest.MessageIndex != 10 {
			t.Fatalf("expected message index 10, got %d", latest.MessageIndex)
		}
	})

	t.Run("GetLatestBySessionID returns ErrNotFound when empty", func(t *testing.T) {
		_, err := repo.GetLatestBySessionID(ctx, "no-checkpoints-here")
		assertErrCode(t, err, types.ErrCodeNotFound)
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustNewSession(t *testing.T) *session.Session {
	t.Helper()
	s, err := session.NewSession("contract-test", "user-1", "device-1")
	if err != nil {
		t.Fatalf("mustNewSession: %v", err)
	}
	return s
}

func mustNewMessage(t *testing.T, sessionID string) *session.Message {
	t.Helper()
	m, err := session.NewMessage(sessionID, "hello from contract test", session.RoleUser)
	if err != nil {
		t.Fatalf("mustNewMessage: %v", err)
	}
	return m
}

func mustNewCheckpoint(t *testing.T, sessionID string, index int) *session.Checkpoint {
	t.Helper()
	cp, err := session.NewCheckpoint(sessionID, index, "checkpoint summary")
	if err != nil {
		t.Fatalf("mustNewCheckpoint: %v", err)
	}
	return cp
}

func assertErrCode(t *testing.T, err error, code types.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %s, got nil", code)
	}
	var appErr *types.AppError
	if !types.As(err, &appErr) {
		t.Fatalf("expected *types.AppError, got %T: %v", err, err)
	}
	if appErr.Code != code {
		t.Fatalf("expected error code %s, got %s: %s", code, appErr.Code, appErr.Message)
	}
}
