// Package export writes the atlas as JSON, Markdown, and a self-contained HTML report.
package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Harry0052/gridatlas/internal/model"
)

// WriteDir writes atlas.json, summary.md, regions.geojson, and report.html.
// script is the compiled map program. template is the HTML shell.
func WriteDir(dir string, atlas *model.Atlas, geo []byte, template string, script []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := Marshal(atlas)
	if err != nil {
		return err
	}
	summary := Summary(atlas)
	if err := writeFile(dir, "atlas.json", raw); err != nil {
		return err
	}
	if err := writeFile(dir, "summary.md", []byte(summary)); err != nil {
		return err
	}
	if err := writeFile(dir, "regions.geojson", geo); err != nil {
		return err
	}
	html, err := ReportHTML(template, raw, geo, []byte(summary), script)
	if err != nil {
		return err
	}
	return writeFile(dir, "report.html", html)
}

func writeFile(dir, name string, data []byte) error {
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || name != filepath.Base(name) {
		return fmt.Errorf("refusing to write %s", name)
	}
	path := filepath.Join(dir, name)
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to write through symlink %s", name)
	}
	return os.WriteFile(path, data, 0o644)
}

// Marshal encodes an atlas with HTML-sensitive characters escaped.
func Marshal(atlas *model.Atlas) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	enc.SetIndent("", "  ")
	if err := enc.Encode(atlas); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ReportHTML inlines the atlas, geometry, summary, and script into the shell.
func ReportHTML(template string, atlas, geo, summary, script []byte) ([]byte, error) {
	for _, part := range []struct {
		name string
		body []byte
	}{
		{"atlas", atlas},
		{"geo", geo},
		{"summary", summary},
		{"script", script},
	} {
		if bytes.Contains(part.body, []byte("__GRIDATLAS_")) {
			return nil, fmt.Errorf("%s contains a template token", part.name)
		}
	}
	geoJSON, err := jsonString(geo)
	if err != nil {
		return nil, err
	}
	summaryJSON, err := json.Marshal(string(summary))
	if err != nil {
		return nil, err
	}
	html := template
	replacements := []struct {
		token string
		value string
	}{
		{"__GRIDATLAS_ATLAS__", string(bytes.TrimSpace(atlas))},
		{"__GRIDATLAS_GEO__", string(geoJSON)},
		{"__GRIDATLAS_SUMMARY__", string(summaryJSON)},
		{"__GRIDATLAS_SCRIPT__", string(script)},
	}
	for _, item := range replacements {
		if !strings.Contains(html, item.token) {
			return nil, fmt.Errorf("template is missing %s", item.token)
		}
		html = strings.Replace(html, item.token, item.value, 1)
	}
	return []byte(html), nil
}

func jsonString(raw []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("geometry is not valid JSON")
	}
	out, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Summary is the human-readable report.
func Summary(atlas *model.Atlas) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# GridAtlas report\n\n")
	fmt.Fprintf(&b, "Schema: %s\n\n", atlas.SchemaVersion)
	fmt.Fprintf(&b, "As of: %s\n\n", atlas.AsOf)
	fmt.Fprintf(&b, "## Limitations\n\n")
	for _, line := range atlas.Limitations {
		fmt.Fprintf(&b, "- %s\n", line)
	}
	b.WriteString("\n")
	for _, region := range atlas.Regions {
		fmt.Fprintf(&b, "## %s — %s\n\n", region.RegionID, region.Name)
		writeMetric(&b, "Installed nameplate MW", region.Metrics.InstalledNameplateMW)
		writeMetric(&b, "Queued requested MW", region.Metrics.QueuedRequestedMW)
		writeHeadroom(&b, region.Metrics.HeadroomMW)
	}
	b.WriteString("## How to trace one finding\n\n")
	b.WriteString("Pick a metric section. The snapshot file and SHA-256 point at the fixture rows. Included trace lines are the only rows in the value. Duplicate, withdrawn, omitted, and ambiguous lines are not added.\n")
	return b.String()
}

func writeMetric(b *strings.Builder, title string, metric model.Metric) {
	fmt.Fprintf(b, "### %s\n\n", title)
	fmt.Fprintf(b, "- Status: %s\n", metric.DisplayStatus)
	fmt.Fprintf(b, "- Value: %s\n", valueText(metric.DisplayStatus, metric.ValueMW, metric.Zero))
	if metric.ReleaseDate != "" {
		fmt.Fprintf(b, "- Release date: %s\n", metric.ReleaseDate)
	}
	if metric.ReportingPeriod != "" {
		fmt.Fprintf(b, "- Reporting period: %s\n", metric.ReportingPeriod)
	}
	if metric.AgeDays != nil {
		fmt.Fprintf(b, "- Age: %d days\n", *metric.AgeDays)
	}
	if metric.FreshnessThresholdDays != nil {
		fmt.Fprintf(b, "- Freshness threshold: %d days\n", *metric.FreshnessThresholdDays)
	}
	if metric.Reason != "" {
		fmt.Fprintf(b, "- Reason: %s\n", metric.Reason)
	}
	if metric.SourceUnit != "" {
		fmt.Fprintf(b, "- Source unit: %s\n", metric.SourceUnit)
	}
	fmt.Fprintf(b, "- Definition: %s\n", metric.Definition)
	if metric.Provenance.SourceURL != "" {
		fmt.Fprintf(b, "- Source: %s\n", metric.Provenance.SourceURL)
	}
	if metric.Provenance.SnapshotFile != "" {
		fmt.Fprintf(b, "- Snapshot: %s\n", metric.Provenance.SnapshotFile)
	}
	if metric.Provenance.ContentSHA256 != "" {
		fmt.Fprintf(b, "- SHA-256: %s\n", metric.Provenance.ContentSHA256)
	}
	if metric.Provenance.MappingMethod != "" {
		fmt.Fprintf(b, "- Mapping: %s\n", metric.Provenance.MappingMethod)
	}
	fmt.Fprintf(b, "- Coverage: included %d, duplicate rows ignored %d, omitted %d, withdrawn %d, ambiguous rows %d, ambiguous requests %d\n",
		metric.Coverage.IncludedRecords,
		metric.Coverage.DuplicateRowsIgnored,
		metric.Coverage.OmittedRows,
		metric.Coverage.WithdrawnRows,
		metric.Coverage.AmbiguousRows,
		metric.Coverage.AmbiguousRequests,
	)
	if len(metric.Coverage.IncludedRecordIDs) > 0 {
		fmt.Fprintf(b, "- Included records: %s\n", strings.Join(metric.Coverage.IncludedRecordIDs, ", "))
	}
	if len(metric.Trace) > 0 {
		b.WriteString("- Trace:\n")
		for _, item := range metric.Trace {
			request := item.RequestID
			if request == "" {
				request = "-"
			}
			unit := item.Unit
			if unit == "" {
				unit = "MW"
			}
			fmt.Fprintf(b, "  - %s request %s %s %d %s %s\n", item.RecordID, request, item.Technology, item.QuantityMW, unit, item.Disposition)
		}
	}
	b.WriteString("\n")
}

func writeHeadroom(b *strings.Builder, item model.Headroom) {
	b.WriteString("### Headroom\n\n")
	fmt.Fprintf(b, "- Status: %s\n", item.DisplayStatus)
	fmt.Fprintf(b, "- Value: %s\n", valueText(item.DisplayStatus, item.ValueMW, item.Zero))
	fmt.Fprintf(b, "- Reason: %s\n", item.Reason)
	fmt.Fprintf(b, "- Definition: %s\n\n", item.Definition)
}

func valueText(status string, value *int, zero bool) string {
	if value == nil {
		switch status {
		case model.DisplayUnknown:
			return "unknown"
		case model.StatusUnobserved:
			return "unobserved"
		case model.StatusMissing:
			return "missing"
		case model.StatusIncompatible:
			return "incompatible"
		default:
			return status
		}
	}
	if zero {
		return fmt.Sprintf("%d MW (reported zero)", *value)
	}
	return fmt.Sprintf("%d MW", *value)
}
