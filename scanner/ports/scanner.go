package ports

import (
	"context"

	"github.com/SecDuckOps/shared/scanner/domain"
)

// ScannerServicePort is the single interface the MasterAgent uses to run scans.
// Previously backed by DockerWarden + parsers.
// Now backed by MCP tool calls — the implementation lives in the agent's MCPScannerAdapter.
type ScannerServicePort interface {
	// RunScan executes one named scanner against a target and returns findings.
	RunScan(ctx context.Context, target string, scannerName string) (domain.ScanResult, error)

	// RunScanBatch executes multiple scanners in parallel.
	RunScanBatch(ctx context.Context, target string, scannerNames []string) []ScanBatchResult

	// HasScanner reports whether the named scanner is available.
	HasScanner(scannerName string) bool

	// AvailableScanners returns the names of all scanners currently available.
	AvailableScanners() []string
}

// ScanBatchResult holds the result of one scanner in a batch run.
type ScanBatchResult struct {
	ScannerName string
	Result      domain.ScanResult
	Err         error
}
