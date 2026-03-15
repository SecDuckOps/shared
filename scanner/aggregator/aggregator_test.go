package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockScannerPort is a mock of ScannerPort
type MockScannerPort struct {
	mock.Mock
}

func (m *MockScannerPort) RunScan(ctx context.Context, opts ports.ScanOpts) (domain.ScanResult, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(domain.ScanResult), args.Error(1)
}

func (m *MockScannerPort) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockParser is a mock of ResultParserPort
type MockParser struct {
	mock.Mock
}

func (m *MockParser) Parse(raw []byte) ([]domain.Finding, error) {
	args := m.Called(raw)
	return args.Get(0).([]domain.Finding), args.Error(1)
}

func (m *MockParser) ScannerName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockParser) SupportedFormats() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func TestScannerService_RunScan(t *testing.T) {
	ctx := context.Background()

	t.Run("successful scan and parse", func(t *testing.T) {
		mockWarden := new(MockScannerPort)
		mockParser := new(MockParser)

		mockParser.On("ScannerName").Return("test-scanner")
		
		svc := NewScannerService(mockWarden, []ports.ResultParserPort{mockParser})

		opts := ports.ScanOpts{TargetDir: "/tmp/code", Scanner: "test-scanner"}
		expectedResult := domain.ScanResult{
			ScanID: "test-id",
			Target: `{"mock":"json"}`, // Hacked stdout
			StartTime: time.Now(),
		}

		mockWarden.On("RunScan", ctx, opts).Return(expectedResult, nil)

		expectedFindings := []domain.Finding{
			{ID: "VULN-1", Title: "Mock Vuln"},
		}
		mockParser.On("Parse", []byte(`{"mock":"json"}`)).Return(expectedFindings, nil)

		res, err := svc.RunScan(ctx, "/tmp/code", "test-scanner")
		assert.NoError(t, err)
		assert.Equal(t, "/tmp/code", res.Target) // Target should be restored
		assert.Len(t, res.Findings, 1)
		assert.Equal(t, "VULN-1", res.Findings[0].ID)

		mockWarden.AssertExpectations(t)
		mockParser.AssertExpectations(t)
	})

	t.Run("scanner not found", func(t *testing.T) {
		svc := NewScannerService(new(MockScannerPort), nil)
		_, err := svc.RunScan(ctx, "/tmp/code", "unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no parser registered")
	})

	t.Run("warden execution fails", func(t *testing.T) {
		mockWarden := new(MockScannerPort)
		mockParser := new(MockParser)
		mockParser.On("ScannerName").Return("test-scanner")
		svc := NewScannerService(mockWarden, []ports.ResultParserPort{mockParser})

		opts := ports.ScanOpts{TargetDir: "/tmp/code", Scanner: "test-scanner"}
		mockWarden.On("RunScan", ctx, opts).Return(domain.ScanResult{}, errors.New("docker error"))

		_, err := svc.RunScan(ctx, "/tmp/code", "test-scanner")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scan execution failed")
	})

	t.Run("parser fails", func(t *testing.T) {
		mockWarden := new(MockScannerPort)
		mockParser := new(MockParser)

		mockParser.On("ScannerName").Return("test-scanner")
		svc := NewScannerService(mockWarden, []ports.ResultParserPort{mockParser})

		opts := ports.ScanOpts{TargetDir: "/tmp/code", Scanner: "test-scanner"}
		expectedResult := domain.ScanResult{
			Target: "invalid json",
		}

		mockWarden.On("RunScan", ctx, opts).Return(expectedResult, nil)
		mockParser.On("Parse", []byte("invalid json")).Return([]domain.Finding{}, errors.New("parse error"))

		res, err := svc.RunScan(ctx, "/tmp/code", "test-scanner")
		
		// It shouldn't return a Go error, but rather embed the error in the ScanResult
		assert.NoError(t, err)
		assert.Contains(t, res.Error, "Failed to parse output")
		assert.Contains(t, res.Error, "parse error")
		assert.Empty(t, res.Findings)
	})
}
