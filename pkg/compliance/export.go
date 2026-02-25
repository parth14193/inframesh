// compliance_export.go — machine-readable export for Report.
// Adds ToJSON() and ToCSV() methods to enable automated pipelines
// and `infracore compliance audit --output=json` support.
package compliance

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// OutputFormat specifies the serialisation format for a compliance report.
type OutputFormat string

const (
	FormatASCII OutputFormat = "ascii" // human-readable terminal output (default)
	FormatJSON  OutputFormat = "json"  // machine-readable JSON
	FormatCSV   OutputFormat = "csv"   // spreadsheet-friendly CSV
)

// ParseOutputFormat converts a CLI flag string to an OutputFormat.
// Returns an error for unrecognised values.
func ParseOutputFormat(s string) (OutputFormat, error) {
	switch strings.ToLower(s) {
	case "ascii", "":
		return FormatASCII, nil
	case "json":
		return FormatJSON, nil
	case "csv":
		return FormatCSV, nil
	default:
		return FormatASCII, fmt.Errorf("unknown output format %q — valid: ascii, json, csv", s)
	}
}

// Export serialises the report into the requested format.
func (r *Report) Export(format OutputFormat) (string, error) {
	switch format {
	case FormatJSON:
		return r.ToJSON()
	case FormatCSV:
		return r.ToCSV()
	default:
		return r.Render(), nil
	}
}

// ToJSON serialises a Report to indented JSON.
func (r *Report) ToJSON() (string, error) {
	// Attach export metadata to the report for programmatic consumers.
	type jsonReport struct {
		Framework   Framework     `json:"framework"`
		Timestamp   time.Time     `json:"timestamp"`
		ScorePct    float64       `json:"score_pct"`
		TotalChecks int           `json:"total_checks"`
		Passed      int           `json:"passed"`
		Failed      int           `json:"failed"`
		Warnings    int           `json:"warnings"`
		Skipped     int           `json:"skipped"`
		Results     []CheckResult `json:"results"`
	}

	out := jsonReport{
		Framework:   r.Framework,
		Timestamp:   r.Timestamp,
		ScorePct:    r.Score,
		TotalChecks: r.TotalChecks,
		Passed:      r.Passed,
		Failed:      r.Failed,
		Warnings:    r.Warnings,
		Skipped:     r.Skipped,
		Results:     r.Results,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("compliance: marshal JSON: %w", err)
	}
	return string(data), nil
}

// ToCSV serialises a Report to RFC 4180 CSV.
// Columns: id, title, status, severity, details, remediation
func (r *Report) ToCSV() (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Header
	if err := w.Write([]string{
		"framework", "id", "title", "status", "severity", "details", "remediation",
	}); err != nil {
		return "", fmt.Errorf("compliance: write CSV header: %w", err)
	}

	for _, result := range r.Results {
		row := []string{
			string(r.Framework),
			result.ID,
			result.Title,
			string(result.Status),
			string(result.Severity),
			result.Details,
			result.Remediation,
		}
		if err := w.Write(row); err != nil {
			return "", fmt.Errorf("compliance: write CSV row: %w", err)
		}
	}

	// Summary row
	if err := w.Write([]string{
		"", "SUMMARY", string(r.Framework),
		"", "", fmt.Sprintf("score=%.1f%% passed=%s failed=%s",
			r.Score, strconv.Itoa(r.Passed), strconv.Itoa(r.Failed)),
		"",
	}); err != nil {
		return "", fmt.Errorf("compliance: write CSV summary: %w", err)
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("compliance: flush CSV: %w", err)
	}
	return buf.String(), nil
}
