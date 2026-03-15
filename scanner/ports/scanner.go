package ports

import (
	"context"

	"github.com/SecDuckOps/shared/scanner/domain"
)

// ScanOpts configures a containerized scan run
type ScanOpts struct {
	TargetDir string
	Scanner   string   // Scanner identifier e.g., "trivy", "semgrep"
	Image     string   // Docker image
	Cmd       []string // Override default command if needed
	Env       []string // Environment variables (e.g. TRIVY_QUIET=true)
}

// ScannerPort is implemented by adapters like DockerWarden to run isolated scans
type ScannerPort interface {
	RunScan(ctx context.Context, opts ScanOpts) (domain.ScanResult, error)
	HealthCheck(ctx context.Context) error
}
