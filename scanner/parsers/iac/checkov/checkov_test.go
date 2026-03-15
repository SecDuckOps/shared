package checkov

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckovParser_Parse(t *testing.T) {
	parser := NewCheckovParser()

	assert.Equal(t, "checkov", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid checkov json object", func(t *testing.T) {
		raw := []byte(`{
			"results": {
				"failed_checks": [
					{
						"check_id": "CKV_AWS_20",
						"check_name": "S3 Bucket has an ACL defined which allows public READ access.",
						"file_path": "/main.tf",
						"file_line_range": [1, 5],
						"resource": "aws_s3_bucket.foo",
						"severity": "HIGH"
					}
				]
			}
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "CKV_AWS_20", f.ID)
		assert.Equal(t, "S3 Bucket has an ACL defined which allows public READ access.", f.Title)
		assert.Equal(t, domain.SeverityHigh, f.Severity)
		assert.Equal(t, domain.ScannerTypeIaC, f.Type)
		assert.Equal(t, "/main.tf", f.File)
		assert.Equal(t, 1, f.Line)
		assert.Equal(t, "checkov", f.Scanner)
		assert.Equal(t, "aws_s3_bucket.foo", f.Match)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse checkov json")
	})
}
