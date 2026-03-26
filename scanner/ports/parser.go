package ports

import (
	"github.com/SecDuckOps/shared/scanner/domain"
)

// ResultParserPort is implemented by individual tool parsers
type ResultParserPort interface {
	Parse(raw []byte) ([]domain.Finding, error)
	ScannerName() string
	ScannerType() domain.ScannerType
	SupportedFormats() []string
	GetScanCommand(target string) []string
}
