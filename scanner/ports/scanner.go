package ports

import (
	"context"

	"github.com/SecDuckOps/shared/scanner/domain"
)

// ScanOpts configures a containerized scan run
type ScanOpts struct {
	TargetDir   string
	Scanner     string             // Scanner identifier e.g., "trivy", "semgrep"
	ScannerType domain.ScannerType // Scanner category e.g., "SAST", "CONTAINER"
	Image       string             // Docker image
	Cmd         []string           // Override default command if needed
	Env         []string           // Environment variables (e.g. TRIVY_QUIET=true)
}

// ScannerPort is implemented by adapters like DockerWarden to run isolated scans
type ScannerPort interface {
	RunScan(ctx context.Context, opts ScanOpts) (domain.ScanResult, error)
	HealthCheck(ctx context.Context) error
}

// ScannerServicePort is the single interface the MasterAgent uses to run scans.
type ScannerServicePort interface {
	RunScan(ctx context.Context, target string, scannerName string) (domain.ScanResult, error)
	RunScanBatch(ctx context.Context, target string, scannerNames []string) []ScanBatchResult
	HasScanner(scannerName string) bool
	AvailableScanners() []string
}

// ScanBatchResult holds the result of one scanner in a batch run.
type ScanBatchResult struct {
	ScannerName string
	Result      domain.ScanResult
	Err         error
}
