package kics

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKicsParser_Parse(t *testing.T) {
	parser := NewKicsParser()

	assert.Equal(t, "kics", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid kics json", func(t *testing.T) {
		raw := []byte(`{
			"queries": [
				{
					"query_name": "S3 Bucket Without Versioning",
					"query_id": "e38a8e0a",
					"severity": "MEDIUM",
					"description": "Amazon S3 bucket should have versioning enabled",
					"files": [
						{
							"file_name": "main.tf",
							"line": 5,
							"expected_value": "aws_s3_bucket.bucket.versioning is set to true",
							"actual_value": "aws_s3_bucket.bucket.versioning is missing"
						}
					]
				}
			]
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "e38a8e0a", f.ID)
		assert.Equal(t, "S3 Bucket Without Versioning", f.Title)
		assert.Equal(t, domain.SeverityMedium, f.Severity)
		assert.Equal(t, domain.ScannerTypeIaC, f.Type)
		assert.Equal(t, "main.tf", f.File)
		assert.Equal(t, 5, f.Line)
		assert.Equal(t, "kics", f.Scanner)
		assert.Equal(t, "Expected: aws_s3_bucket.bucket.versioning is set to true | Actual: aws_s3_bucket.bucket.versioning is missing", f.Match)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse kics json")
	})
}
