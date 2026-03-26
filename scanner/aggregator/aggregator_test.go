package aggregator

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

type fakeScannerPort struct {
	lastOpts ports.ScanOpts
	result   domain.ScanResult
	err      error
}

func (f *fakeScannerPort) RunScan(_ context.Context, opts ports.ScanOpts) (domain.ScanResult, error) {
	f.lastOpts = opts
	return f.result, f.err
}

func (f *fakeScannerPort) HealthCheck(context.Context) error {
	return nil
}

type fakeParser struct {
	name         string
	scannerType  domain.ScannerType
	commandCalls []string
	findings     []domain.Finding
	parseErr     error
}

func (f *fakeParser) Parse([]byte) ([]domain.Finding, error) {
	if f.parseErr != nil {
		return nil, f.parseErr
	}
	return f.findings, nil
}

func (f *fakeParser) ScannerName() string {
	return f.name
}

func (f *fakeParser) ScannerType() domain.ScannerType {
	return f.scannerType
}

func (f *fakeParser) SupportedFormats() []string {
	return []string{"json"}
}

func (f *fakeParser) GetScanCommand(target string) []string {
	f.commandCalls = append(f.commandCalls, target)
	return []string{"scan", target}
}

func TestRunScanUsesRequestedTargetAndParserMetadata(t *testing.T) {
	parser := &fakeParser{
		name:        "semgrep",
		scannerType: domain.ScannerTypeSAST,
		findings: []domain.Finding{
			{ID: "finding-1", Title: "issue"},
		},
	}
	scanner := &fakeScannerPort{
		result: domain.ScanResult{RawOutput: `{"ok":true}`},
	}
	service := NewScannerService(scanner, []ports.ResultParserPort{parser})

	result, err := service.RunScan(context.Background(), "/workspace/project", "semgrep")
	if err != nil {
		t.Fatalf("RunScan returned error: %v", err)
	}

	if scanner.lastOpts.TargetDir != "/workspace/project" {
		t.Fatalf("expected target dir to propagate, got %q", scanner.lastOpts.TargetDir)
	}
	if scanner.lastOpts.ScannerType != domain.ScannerTypeSAST {
		t.Fatalf("expected scanner type to propagate, got %q", scanner.lastOpts.ScannerType)
	}
	if !reflect.DeepEqual(scanner.lastOpts.Cmd, []string{"scan", "/workspace/project"}) {
		t.Fatalf("expected command to use real target, got %#v", scanner.lastOpts.Cmd)
	}
	if !reflect.DeepEqual(parser.commandCalls, []string{"/workspace/project"}) {
		t.Fatalf("expected parser command to be built from target, got %#v", parser.commandCalls)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected parsed findings to be attached, got %#v", result.Findings)
	}
}

func TestRunScanPreservesRawResultOnParseFailure(t *testing.T) {
	parser := &fakeParser{
		name:        "trivy",
		scannerType: domain.ScannerTypeDependency,
		parseErr:    errors.New("bad json"),
	}
	scanner := &fakeScannerPort{
		result: domain.ScanResult{RawOutput: `{"broken":true}`},
	}
	service := NewScannerService(scanner, []ports.ResultParserPort{parser})

	result, err := service.RunScan(context.Background(), "/repo", "trivy")
	if err != nil {
		t.Fatalf("RunScan returned error: %v", err)
	}
	if result.Error == "" {
		t.Fatal("expected parse failure to be attached to result")
	}
	if result.RawOutput != `{"broken":true}` {
		t.Fatalf("expected raw output to be preserved, got %q", result.RawOutput)
	}
}

func TestRunScanBatchAndAvailableScanners(t *testing.T) {
	trivy := &fakeParser{name: "trivy", scannerType: domain.ScannerTypeDependency}
	semgrep := &fakeParser{name: "semgrep", scannerType: domain.ScannerTypeSAST}
	scanner := &fakeScannerPort{result: domain.ScanResult{RawOutput: ""}}
	service := NewScannerService(scanner, []ports.ResultParserPort{trivy, semgrep})

	if !service.HasScanner("trivy") {
		t.Fatal("expected HasScanner(trivy) to be true")
	}
	if service.HasScanner("nuclei") {
		t.Fatal("expected HasScanner(nuclei) to be false")
	}

	if got := service.AvailableScanners(); !reflect.DeepEqual(got, []string{"semgrep", "trivy"}) {
		t.Fatalf("expected sorted scanners, got %#v", got)
	}

	batch := service.RunScanBatch(context.Background(), "/repo", []string{"trivy", "missing"})
	if len(batch) != 2 {
		t.Fatalf("expected two batch results, got %d", len(batch))
	}
	if batch[0].Err != nil {
		t.Fatalf("expected trivy scan to succeed, got %v", batch[0].Err)
	}
	if batch[1].Err == nil {
		t.Fatal("expected missing parser to surface as error")
	}
}
