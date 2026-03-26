package aggregator

import (
	"context"
	"fmt"
	"slices"

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
	parser, err := s.parserFor(scannerName)
	if err != nil {
		return domain.ScanResult{}, err
	}

	opts := buildScanOptions(target, scannerName, parser)
	rawResult, err := s.warden.RunScan(ctx, opts)
	if err != nil {
		return rawResult, fmt.Errorf("scan execution failed: %w", err)
	}

	return parseScanResult(rawResult, parser)
}

func (s *ScannerService) RunScanBatch(ctx context.Context, target string, scannerNames []string) []ports.ScanBatchResult {
	results := make([]ports.ScanBatchResult, 0, len(scannerNames))
	for _, scannerName := range scannerNames {
		result, err := s.RunScan(ctx, target, scannerName)
		results = append(results, ports.ScanBatchResult{
			ScannerName: scannerName,
			Result:      result,
			Err:         err,
		})
	}
	return results
}

func (s *ScannerService) HasScanner(scannerName string) bool {
	_, exists := s.parsers[scannerName]
	return exists
}

func (s *ScannerService) AvailableScanners() []string {
	names := make([]string, 0, len(s.parsers))
	for name := range s.parsers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func (s *ScannerService) parserFor(scannerName string) (ports.ResultParserPort, error) {
	parser, exists := s.parsers[scannerName]
	if !exists {
		return nil, fmt.Errorf("no parser registered for scanner: %s", scannerName)
	}
	return parser, nil
}

func buildScanOptions(target, scannerName string, parser ports.ResultParserPort) ports.ScanOpts {
	return ports.ScanOpts{
		TargetDir:   target,
		Scanner:     scannerName,
		ScannerType: parser.ScannerType(),
		Cmd:         parser.GetScanCommand(target),
	}
}

func parseScanResult(rawResult domain.ScanResult, parser ports.ResultParserPort) (domain.ScanResult, error) {
	rawOutput := []byte(rawResult.RawOutput)
	if len(rawOutput) == 0 {
		return rawResult, nil
	}

	findings, err := parser.Parse(rawOutput)
	if err != nil {
		rawResult.Error = fmt.Sprintf("Failed to parse output: %v. Original output size: %d bytes.", err, len(rawOutput))
		rawResult.RawOutput = string(rawOutput)
		return rawResult, nil
	}

	rawResult.Findings = findings
	return rawResult, nil
}
