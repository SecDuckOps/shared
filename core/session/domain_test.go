package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/SecDuckOps/shared/types"
)

// ---------------------------------------------------------------------------
// NewSession
// ---------------------------------------------------------------------------

func TestNewSession_Valid(t *testing.T) {
	s, err := NewSession("my-session", "user-1", "device-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if s.Status != SessionStatusActive {
		t.Fatalf("expected status %s, got %s", SessionStatusActive, s.Status)
	}
	if s.SyncStatus != SyncStatusLocalOnly {
		t.Fatalf("expected sync status %s, got %s", SyncStatusLocalOnly, s.SyncStatus)
	}
	if s.Version != 1 {
		t.Fatalf("expected version 1, got %d", s.Version)
	}
}

func TestNewSession_EmptyName(t *testing.T) {
	_, err := NewSession("", "user-1", "device-1")
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestNewSession_EmptyUserID(t *testing.T) {
	_, err := NewSession("name", "", "device-1")
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestNewSession_EmptyDeviceID(t *testing.T) {
	_, err := NewSession("name", "user-1", "")
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestNewSession_NameTooLong(t *testing.T) {
	longName := strings.Repeat("x", MaxSessionNameLen+1)
	_, err := NewSession(longName, "user-1", "device-1")
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestNewSession_UserIDTooLong(t *testing.T) {
	longID := strings.Repeat("x", MaxIdentifierLen+1)
	_, err := NewSession("name", longID, "device-1")
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestNewSession_DeviceIDTooLong(t *testing.T) {
	longID := strings.Repeat("x", MaxIdentifierLen+1)
	_, err := NewSession("name", "user-1", longID)
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

// ---------------------------------------------------------------------------
// Close & SoftDelete — state transition guards
// ---------------------------------------------------------------------------

func TestClose_ActiveSession(t *testing.T) {
	s := mustSession(t)
	if err := s.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status != SessionStatusClosed {
		t.Fatalf("expected closed, got %s", s.Status)
	}
	if s.Version != 2 {
		t.Fatalf("expected version 2, got %d", s.Version)
	}
}

func TestClose_ClosedSession(t *testing.T) {
	s := mustSession(t)
	_ = s.Close()
	err := s.Close()
	assertAppError(t, err, types.ErrCodeInvalidTransition)
}

func TestClose_DeletedSession(t *testing.T) {
	s := mustSession(t)
	_ = s.SoftDelete()
	err := s.Close()
	assertAppError(t, err, types.ErrCodeInvalidTransition)
}

func TestSoftDelete_ActiveSession(t *testing.T) {
	s := mustSession(t)
	if err := s.SoftDelete(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status != SessionStatusDeleted {
		t.Fatalf("expected deleted, got %s", s.Status)
	}
	if s.DeletedAt == nil {
		t.Fatal("expected DeletedAt to be set")
	}
}

func TestSoftDelete_ClosedSession(t *testing.T) {
	s := mustSession(t)
	_ = s.Close()
	if err := s.SoftDelete(); err != nil {
		t.Fatalf("unexpected error: soft delete from closed should be allowed: %v", err)
	}
}

func TestSoftDelete_AlreadyDeleted(t *testing.T) {
	s := mustSession(t)
	_ = s.SoftDelete()
	err := s.SoftDelete()
	assertAppError(t, err, types.ErrCodeInvalidTransition)
}

// ---------------------------------------------------------------------------
// SetMetadata — encapsulation
// ---------------------------------------------------------------------------

func TestSetMetadata_Valid(t *testing.T) {
	s := mustSession(t)
	if err := s.SetMetadata("env", "production"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Metadata["env"] != "production" {
		t.Fatal("expected metadata to be set")
	}
}

func TestSetMetadata_EmptyKey(t *testing.T) {
	s := mustSession(t)
	err := s.SetMetadata("", "val")
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestSetMetadata_KeyTooLong(t *testing.T) {
	s := mustSession(t)
	err := s.SetMetadata(strings.Repeat("k", MaxMetadataKeyLen+1), "val")
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestSetMetadata_ValueTooLong(t *testing.T) {
	s := mustSession(t)
	err := s.SetMetadata("key", strings.Repeat("v", MaxMetadataValueLen+1))
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestSetMetadata_TooManyEntries(t *testing.T) {
	s := mustSession(t)
	for i := 0; i < MaxMetadataEntries; i++ {
		if err := s.SetMetadata(fmt.Sprintf("key-%d", i), "val"); err != nil {
			t.Fatalf("unexpected error at entry %d: %v", i, err)
		}
	}
	err := s.SetMetadata("one-too-many", "val")
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestSetMetadata_UpdateExistingAtLimit(t *testing.T) {
	s := mustSession(t)
	for i := 0; i < MaxMetadataEntries; i++ {
		_ = s.SetMetadata(fmt.Sprintf("key-%d", i), "val")
	}
	// updating an existing key should still work
	if err := s.SetMetadata("key-0", "updated"); err != nil {
		t.Fatalf("updating existing key at limit should succeed: %v", err)
	}
	if s.Metadata["key-0"] != "updated" {
		t.Fatal("expected metadata to be updated")
	}
}

// SyncStatus transitions
// ---------------------------------------------------------------------------

func TestTransitionSync_ValidPath(t *testing.T) {
	s := mustSession(t) // starts at local_only

	if err := s.MarkPending(); err != nil {
		t.Fatalf("local_only → pending_sync failed: %v", err)
	}
	if err := s.MarkSynced(); err != nil {
		t.Fatalf("pending_sync → synced failed: %v", err)
	}
	if s.SyncStatus != SyncStatusSynced {
		t.Fatalf("expected synced, got %s", s.SyncStatus)
	}
}

func TestTransitionSync_InvalidPath(t *testing.T) {
	s := mustSession(t) // starts at local_only
	err := s.MarkSynced()
	assertAppError(t, err, types.ErrCodeInvalidTransition)
}

func TestTransitionSync_ConflictRecovery(t *testing.T) {
	s := mustSession(t)
	_ = s.MarkPending()
	_ = s.MarkConflict()
	if err := s.MarkPending(); err != nil {
		t.Fatalf("conflict → pending_sync should be allowed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// NewMessage
// ---------------------------------------------------------------------------

func TestNewMessage_Valid(t *testing.T) {
	m, err := NewMessage("sess-1", "hello", RoleUser)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Content != "hello" {
		t.Fatal("expected content to be hello")
	}
}

func TestNewMessage_EmptyContent(t *testing.T) {
	_, err := NewMessage("sess-1", "", RoleUser)
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestNewMessage_ContentTooLong(t *testing.T) {
	longContent := strings.Repeat("x", MaxMessageLen+1)
	_, err := NewMessage("sess-1", longContent, RoleUser)
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestNewMessage_InvalidRole(t *testing.T) {
	_, err := NewMessage("sess-1", "hello", Role("hacker"))
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestNewMessage_SessionIDTooLong(t *testing.T) {
	_, err := NewMessage(strings.Repeat("s", MaxIdentifierLen+1), "hello", RoleUser)
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

// ---------------------------------------------------------------------------
// NewCheckpoint
// ---------------------------------------------------------------------------

func TestNewCheckpoint_Valid(t *testing.T) {
	c, err := NewCheckpoint("sess-1", 5, "summary of conversation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.MessageIndex != 5 {
		t.Fatalf("expected index 5, got %d", c.MessageIndex)
	}
}

func TestNewCheckpoint_NegativeIndex(t *testing.T) {
	_, err := NewCheckpoint("sess-1", -1, "summary")
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestNewCheckpoint_SummaryTooLong(t *testing.T) {
	_, err := NewCheckpoint("sess-1", 0, strings.Repeat("x", MaxCheckpointLen+1))
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

// ---------------------------------------------------------------------------
// NewEvent
// ---------------------------------------------------------------------------

func TestNewEvent_Valid(t *testing.T) {
	e, err := NewEvent("sess-1", EventSessionCreated, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Payload == nil {
		t.Fatal("expected payload to be initialized (not nil)")
	}
}

func TestNewEvent_EmptySessionID(t *testing.T) {
	_, err := NewEvent("", EventSessionCreated, nil)
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

func TestNewEvent_EmptyEventType(t *testing.T) {
	_, err := NewEvent("sess-1", "", nil)
	assertAppError(t, err, types.ErrCodeInvalidInput)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustSession(t *testing.T) *Session {
	t.Helper()
	s, err := NewSession("test-session", "user-1", "device-1")
	if err != nil {
		t.Fatalf("mustSession: %v", err)
	}
	return s
}

func assertAppError(t *testing.T, err error, expectedCode types.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*types.AppError)
	if !ok {
		t.Fatalf("expected *types.AppError, got %T: %v", err, err)
	}
	if appErr.Code != expectedCode {
		t.Fatalf("expected error code %s, got %s: %s", expectedCode, appErr.Code, appErr.Message)
	}
}
