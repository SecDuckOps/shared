package ports

import (
	"context"

	"github.com/SecDuckOps/shared/scanner/domain"
)

// ScanStoragePort is implemented by SQLite (Agent) or PostgreSQL (Server) adapters
type ScanStoragePort interface {
	SaveResult(ctx context.Context, result domain.ScanResultRecord) error
	GetResult(ctx context.Context, id string) (domain.ScanResultRecord, error)
	ListResults(ctx context.Context, filter domain.ScanFilter) ([]domain.ScanResultRecord, error)
	SaveFindings(ctx context.Context, scanID string, findings []domain.Finding) error
	GetFindings(ctx context.Context, scanID string) ([]domain.Finding, error)
}
