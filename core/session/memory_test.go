package session_test

import (
	"testing"

	"github.com/SecDuckOps/shared/core/session"
	sessiontest "github.com/SecDuckOps/shared/core/session/testing"
)

func TestMemorySessionRepository_Contract(t *testing.T) {
	store := session.NewMemoryStore()
	sessiontest.RunSessionRepositoryContract(t, store.SessionRepository())
}

func TestMemoryMessageRepository_Contract(t *testing.T) {
	store := session.NewMemoryStore()
	sessiontest.RunMessageRepositoryContract(t, store.MessageRepository())
}

func TestMemoryCheckpointRepository_Contract(t *testing.T) {
	store := session.NewMemoryStore()
	sessiontest.RunCheckpointRepositoryContract(t, store.CheckpointRepository())
}
