package aggregator

import (
	"context"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// ScannerService orchestrates the execution and parsing of security scans.
type ScannerService struct {
	warden  ports.ScannerPort
	parsers map[string]ports.ResultParserPort
}

// NewScannerService creates a new ScannerService with the given scanner port (Warden)
// and a list of registered parsers.
func NewScannerService(warden ports.ScannerPort, parsers []ports.ResultParserPort) *ScannerService {
	pMap := make(map[string]ports.ResultParserPort)
	for _, p := range parsers {
		pMap[p.ScannerName()] = p
	}

	return &ScannerService{
		warden:  warden,
		parsers: pMap,
	}
}

// RunScan executes the scanner via Warden and parses the results.
func (s *ScannerService) RunScan(ctx context.Context, target string, scannerName string) (domain.ScanResult, error) {
	parser, exists := s.parsers[scannerName]
	if !exists {
		return domain.ScanResult{}, fmt.Errorf("no parser registered for scanner: %s", scannerName)
	}

	// 1. Run the scan through Warden
	opts := ports.ScanOpts{
		TargetDir: target,
		Scanner:   scannerName,
		Cmd:       parser.GetScanCommand("."),
	}

	rawResult, err := s.warden.RunScan(ctx, opts)
	if err != nil {
		// We might still have a partial result even on error (e.g. timeout)
		return rawResult, fmt.Errorf("scan execution failed: %w", err)
	}

	// 2. The raw output is currently hacked into the 'Target' field by the DockerWarden.
	// We extract it and parse it.
	// Note: In an ideal world, DockerWarden would return stdout directly in a 'RawOutput' field,
	// but we'll work with the current behavior.
	rawOutput := []byte(rawResult.Target)

	// Since we are fixing the hack, restore Target to its true value
	rawResult.Target = target

	// If there's no output to parse, or the scan actually failed without returning output
	if len(rawOutput) == 0 {
		return rawResult, nil
	}

	// 3. Parse the output using the corresponding parser
	findings, err := parser.Parse(rawOutput)
	if err != nil {
		rawResult.Error = fmt.Sprintf("Failed to parse output: %v. Original output size: %d bytes.", err, len(rawOutput))
		rawResult.RawOutput = string(rawOutput)
		return rawResult, nil // Return the raw result with the parsing error and raw output
	}

	// 4. Attach the parsed findings back to the result
	rawResult.Findings = findings

	return rawResult, nil
}
