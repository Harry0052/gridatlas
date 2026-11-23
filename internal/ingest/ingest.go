// Package ingest reads a local fixture directory into a versioned bundle.
package ingest

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Harry0052/gridatlas/internal/model"
)

// Error is an actionable input failure. Category is schema, bounds, checksum, path, or io.
type Error struct {
	Category string
	File     string
	Message  string
}

func (e *Error) Error() string {
	if e.File != "" {
		return fmt.Sprintf("input invalid (%s): %s: %s", e.Category, e.File, e.Message)
	}
	return fmt.Sprintf("input invalid (%s): %s", e.Category, e.Message)
}

func fail(category, file, format string, args ...any) error {
	return &Error{Category: category, File: file, Message: fmt.Sprintf(format, args...)}
}

var (
	regionIDPattern   = regexp.MustCompile(`^[A-Z][A-Z0-9-]{0,31}$`)
	regionNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ()'-]{0,79}$`)
	methodPattern     = regexp.MustCompile(`^[a-z_]{1,32}$`)
	periodPattern     = regexp.MustCompile(`^[A-Za-z0-9-]{2,32}$`)
	shaPattern        = regexp.MustCompile(`^[0-9a-f]{64}$`)
	idPattern         = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)
	techPattern       = regexp.MustCompile(`^[A-Za-z][A-Za-z-]{0,31}$`)
	unitPattern       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{0,15}$`)
	intPattern        = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
	fixtureURLPattern = regexp.MustCompile(`^fixture://[a-z0-9./_-]+$`)
	httpsURLPattern   = regexp.MustCompile(`^https://[A-Za-z0-9._~:/?#\[\]@!$&'()*+,;=%-]+$`)
)

// Load reads and validates a fixture directory. It does not score coverage.
func Load(dir string) (*model.Bundle, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fail("path", dir, "cannot resolve path")
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fail("io", dir, "cannot read directory")
	}
	if !info.IsDir() {
		return nil, fail("path", dir, "not a directory")
	}

	regions, err := loadRegions(abs)
	if err != nil {
		return nil, err
	}
	crosswalk, err := loadCrosswalk(abs, regions)
	if err != nil {
		return nil, err
	}
	geo, err := loadGeo(abs, regions)
	if err != nil {
		return nil, err
	}
	manifest, snapshots, err := loadManifest(abs, regions)
	if err != nil {
		return nil, err
	}
	return &model.Bundle{
		Regions:   regions,
		Crosswalk: crosswalk,
		Manifest:  manifest,
		Snapshots: snapshots,
		GeoJSON:   geo,
	}, nil
}

func loadRegions(root string) ([]model.Region, error) {
	rows, err := readTable(root, "regions.csv", []string{"region_id", "name"})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fail("schema", "regions.csv", "at least one region is required")
	}
	if len(rows) > model.MaxRegions {
		return nil, fail("bounds", "regions.csv", "region count %d exceeds %d", len(rows), model.MaxRegions)
	}
	seen := map[string]bool{}
	out := make([]model.Region, 0, len(rows))
	for _, row := range rows {
		id := row.values["region_id"]
		name := row.values["name"]
		if !regionIDPattern.MatchString(id) {
			return nil, fail("schema", "regions.csv", "row %d: region_id must match %s", row.line, regionIDPattern.String())
		}
		if !regionNamePattern.MatchString(name) {
			return nil, fail("schema", "regions.csv", "row %d: name is empty or too long", row.line)
		}
		if seen[id] {
			return nil, fail("schema", "regions.csv", "row %d: duplicate region_id %s", row.line, id)
		}
		seen[id] = true
		out = append(out, model.Region{ID: id, Name: name})
	}
	return out, nil
}

func loadCrosswalk(root string, regions []model.Region) ([]model.Crosswalk, error) {
	rows, err := readTable(root, "crosswalk.csv", []string{"source_region", "atlas_region", "mapping_method", "supported"})
	if err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, region := range regions {
		known[region.ID] = true
	}
	out := make([]model.Crosswalk, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		source := row.values["source_region"]
		atlas := row.values["atlas_region"]
		method := row.values["mapping_method"]
		supported := row.values["supported"]
		if !regionIDPattern.MatchString(source) || !regionIDPattern.MatchString(atlas) {
			return nil, fail("schema", "crosswalk.csv", "row %d: region ids must match %s", row.line, regionIDPattern.String())
		}
		if !known[atlas] {
			return nil, fail("schema", "crosswalk.csv", "row %d: atlas_region %s is not in regions.csv", row.line, atlas)
		}
		if !methodPattern.MatchString(method) {
			return nil, fail("schema", "crosswalk.csv", "row %d: mapping_method must be a short lowercase token", row.line)
		}
		var ok bool
		switch supported {
		case "true":
			ok = true
		case "false":
			ok = false
		default:
			return nil, fail("schema", "crosswalk.csv", "row %d: supported must be true or false", row.line)
		}
		key := source + "|" + atlas + "|" + method
		if seen[key] {
			return nil, fail("schema", "crosswalk.csv", "row %d: duplicate crosswalk %s", row.line, key)
		}
		seen[key] = true
		out = append(out, model.Crosswalk{
			SourceRegion: source, AtlasRegion: atlas, MappingMethod: method, Supported: ok,
		})
	}
	return out, nil
}

func loadGeo(root string, regions []model.Region) ([]byte, error) {
	data, err := readFile(root, "regions.geojson")
	if err != nil {
		return nil, err
	}
	var doc struct {
		Type     string `json:"type"`
		Features []struct {
			Type       string `json:"type"`
			Properties struct {
				RegionID     string `json:"region_id"`
				Synthetic    *bool  `json:"synthetic"`
				BoundaryNote string `json:"boundary_note"`
			} `json:"properties"`
			Geometry struct {
				Type        string        `json:"type"`
				Coordinates [][][]float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fail("schema", "regions.geojson", "invalid GeoJSON")
	}
	if doc.Type != "FeatureCollection" {
		return nil, fail("schema", "regions.geojson", "type must be FeatureCollection")
	}
	seen := map[string]bool{}
	for i, feature := range doc.Features {
		if feature.Type != "Feature" || feature.Geometry.Type != "Polygon" {
			return nil, fail("schema", "regions.geojson", "feature %d must be a Polygon feature", i)
		}
		if feature.Properties.Synthetic == nil || !*feature.Properties.Synthetic {
			return nil, fail("schema", "regions.geojson", "feature %d must set synthetic to true", i)
		}
		if !strings.Contains(strings.ToLower(feature.Properties.BoundaryNote), "synthetic") {
			return nil, fail("schema", "regions.geojson", "feature %d boundary_note must say that the shape is synthetic", i)
		}
		if !regionIDPattern.MatchString(feature.Properties.RegionID) {
			return nil, fail("schema", "regions.geojson", "feature %d has an invalid region_id", i)
		}
		if seen[feature.Properties.RegionID] {
			return nil, fail("schema", "regions.geojson", "duplicate feature for %s", feature.Properties.RegionID)
		}
		seen[feature.Properties.RegionID] = true
		if len(feature.Geometry.Coordinates) != 1 {
			return nil, fail("schema", "regions.geojson", "feature %d must have one ring", i)
		}
		ring := feature.Geometry.Coordinates[0]
		if len(ring) < 4 || len(ring) > 64 {
			return nil, fail("bounds", "regions.geojson", "feature %d ring length is out of bounds", i)
		}
		first, last := ring[0], ring[len(ring)-1]
		if len(first) != 2 || first[0] != last[0] || first[1] != last[1] {
			return nil, fail("schema", "regions.geojson", "feature %d ring must be closed", i)
		}
		for _, point := range ring {
			if len(point) != 2 || point[0] < -180 || point[0] > 180 || point[1] < -90 || point[1] > 90 {
				return nil, fail("bounds", "regions.geojson", "feature %d has a coordinate outside lon/lat bounds", i)
			}
		}
	}
	for _, region := range regions {
		if !seen[region.ID] {
			return nil, fail("schema", "regions.geojson", "missing feature for %s", region.ID)
		}
	}
	if len(seen) != len(regions) {
		return nil, fail("schema", "regions.geojson", "features must match regions.csv exactly")
	}
	return data, nil
}

func loadManifest(root string, regions []model.Region) ([]model.ManifestRow, map[string][]model.Measurement, error) {
	rows, err := readTable(root, "manifest.csv", []string{
		"product", "source_url", "release_date", "reporting_period", "unit", "region",
		"mapping_method", "content_sha256", "snapshot_file", "freshness_threshold_days",
	})
	if err != nil {
		return nil, nil, err
	}
	known := map[string]bool{}
	for _, region := range regions {
		known[region.ID] = true
	}
	out := make([]model.ManifestRow, 0, len(rows))
	snapshots := map[string][]model.Measurement{}
	seenKey := map[string]bool{}
	totalRows := 0
	for _, row := range rows {
		product := row.values["product"]
		if product != model.ProductInstalled && product != model.ProductQueued {
			return nil, nil, fail("schema", "manifest.csv", "row %d: product must be %s or %s", row.line, model.ProductInstalled, model.ProductQueued)
		}
		region := row.values["region"]
		if !known[region] {
			return nil, nil, fail("schema", "manifest.csv", "row %d: region %s is not in regions.csv", row.line, region)
		}
		key := region + "|" + product
		if seenKey[key] {
			return nil, nil, fail("schema", "manifest.csv", "row %d: duplicate product %s for %s", row.line, product, region)
		}
		seenKey[key] = true
		sourceURL := row.values["source_url"]
		if !validSourceURL(sourceURL) {
			return nil, nil, fail("schema", "manifest.csv", "row %d: source_url must be a fixture:// or https:// URL without spaces", row.line)
		}
		release, err := time.ParseInLocation("2006-01-02", row.values["release_date"], time.UTC)
		if err != nil {
			return nil, nil, fail("schema", "manifest.csv", "row %d: release_date must be YYYY-MM-DD", row.line)
		}
		period := row.values["reporting_period"]
		if !periodPattern.MatchString(period) {
			return nil, nil, fail("schema", "manifest.csv", "row %d: reporting_period is invalid", row.line)
		}
		unit := row.values["unit"]
		if !unitPattern.MatchString(unit) {
			return nil, nil, fail("schema", "manifest.csv", "row %d: unit is invalid", row.line)
		}
		method := row.values["mapping_method"]
		if !methodPattern.MatchString(method) {
			return nil, nil, fail("schema", "manifest.csv", "row %d: mapping_method is invalid", row.line)
		}
		sum := row.values["content_sha256"]
		if !shaPattern.MatchString(sum) {
			return nil, nil, fail("schema", "manifest.csv", "row %d: content_sha256 must be 64 lowercase hex characters", row.line)
		}
		threshold, err := strconv.Atoi(row.values["freshness_threshold_days"])
		if err != nil || !intPattern.MatchString(row.values["freshness_threshold_days"]) || threshold < 0 || threshold > 3650 {
			return nil, nil, fail("bounds", "manifest.csv", "row %d: freshness_threshold_days must be an integer from 0 to 3650", row.line)
		}
		rel := row.values["snapshot_file"]
		data, err := readFile(root, rel)
		if err != nil {
			return nil, nil, err
		}
		got := sha256.Sum256(data)
		if hex.EncodeToString(got[:]) != sum {
			return nil, nil, fail("checksum", rel, "SHA-256 does not match manifest row %d", row.line)
		}
		measurements, err := parseMeasurements(rel, data)
		if err != nil {
			return nil, nil, err
		}
		totalRows += len(measurements)
		if totalRows > model.MaxRows {
			return nil, nil, fail("bounds", rel, "row count exceeds %d", model.MaxRows)
		}
		snapshots[rel] = measurements
		out = append(out, model.ManifestRow{
			Product:                product,
			SourceURL:              sourceURL,
			ReleaseDate:            release,
			ReportingPeriod:        period,
			Unit:                   unit,
			Region:                 region,
			MappingMethod:          method,
			ContentSHA256:          sum,
			SnapshotFile:           rel,
			FreshnessThresholdDays: threshold,
		})
	}
	return out, snapshots, nil
}

func parseMeasurements(file string, data []byte) ([]model.Measurement, error) {
	rows, err := parseCSV(file, data, []string{
		"record_id", "request_id", "technology", "quantity", "unit", "basis", "row_status",
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fail("schema", file, "snapshot has no data rows")
	}
	if len(rows) > model.MaxRows {
		return nil, fail("bounds", file, "row count exceeds %d", model.MaxRows)
	}
	out := make([]model.Measurement, 0, len(rows))
	for _, row := range rows {
		recordID := row.values["record_id"]
		requestID := row.values["request_id"]
		if !idPattern.MatchString(recordID) {
			return nil, fail("schema", file, "row %d: record_id is invalid", row.line)
		}
		if requestID != "" && !idPattern.MatchString(requestID) {
			return nil, fail("schema", file, "row %d: request_id is invalid", row.line)
		}
		tech := row.values["technology"]
		if !techPattern.MatchString(tech) {
			return nil, fail("schema", file, "row %d: technology is invalid", row.line)
		}
		qtyText := row.values["quantity"]
		if !intPattern.MatchString(qtyText) {
			return nil, fail("schema", file, "row %d: quantity must be a non-negative integer MW value", row.line)
		}
		qty, err := strconv.Atoi(qtyText)
		if err != nil || qty < 0 || qty > model.MaxQuantityMW {
			return nil, fail("bounds", file, "row %d: quantity must be from 0 to %d", row.line, model.MaxQuantityMW)
		}
		unit := row.values["unit"]
		if !unitPattern.MatchString(unit) {
			return nil, fail("schema", file, "row %d: unit is invalid", row.line)
		}
		basis := row.values["basis"]
		if basis != model.BasisNameplate && basis != model.BasisRequested {
			return nil, fail("schema", file, "row %d: basis must be %s or %s", row.line, model.BasisNameplate, model.BasisRequested)
		}
		status := row.values["row_status"]
		switch status {
		case "operating", "active", "withdrawn", "omitted", "ambiguous":
		default:
			return nil, fail("schema", file, "row %d: row_status is not a supported value", row.line)
		}
		out = append(out, model.Measurement{
			RecordID: recordID, RequestID: requestID, Technology: tech,
			Quantity: qty, Unit: unit, Basis: basis, RowStatus: status, Line: row.line,
		})
	}
	return out, nil
}

func validSourceURL(value string) bool {
	if len(value) > 200 {
		return false
	}
	return fixtureURLPattern.MatchString(value) || httpsURLPattern.MatchString(value)
}

type tableRow struct {
	line   int
	values map[string]string
}

func readTable(root, name string, columns []string) ([]tableRow, error) {
	data, err := readFile(root, name)
	if err != nil {
		return nil, err
	}
	return parseCSV(name, data, columns)
}

func parseCSV(file string, data []byte, columns []string) ([]tableRow, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return nil, fail("schema", file, "missing header")
	}
	if len(header) != len(columns) {
		missing := firstMissingColumn(header, columns)
		if missing != "" {
			return nil, fail("schema", file, "missing required column %q", missing)
		}
		return nil, fail("schema", file, "header must be %s", strings.Join(columns, ","))
	}
	for i, want := range columns {
		if strings.TrimSpace(header[i]) != want {
			return nil, fail("schema", file, "missing required column %q", want)
		}
	}
	reader.FieldsPerRecord = len(columns)
	var rows []tableRow
	line := 1
	for {
		line++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fail("schema", file, "row %d: %s", line, csvMessage(err))
		}
		values := make(map[string]string, len(columns))
		for i, column := range columns {
			values[column] = strings.TrimSpace(record[i])
		}
		rows = append(rows, tableRow{line: line, values: values})
	}
	return rows, nil
}

func firstMissingColumn(header, columns []string) string {
	have := map[string]bool{}
	for _, column := range header {
		have[strings.TrimSpace(column)] = true
	}
	for _, column := range columns {
		if !have[column] {
			return column
		}
	}
	return ""
}

func csvMessage(err error) string {
	var parse *csv.ParseError
	if ok := errorAs(err, &parse); ok {
		return parse.Err.Error()
	}
	return "could not parse this row"
}

func errorAs(err error, target **csv.ParseError) bool {
	parsed, ok := err.(*csv.ParseError)
	if !ok {
		return false
	}
	*target = parsed
	return true
}

func readFile(root, name string) ([]byte, error) {
	full, err := safePath(root, name)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(full)
	if err != nil {
		return nil, fail("io", name, "cannot open file")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fail("path", name, "symlinks are not allowed")
	}
	if !info.Mode().IsRegular() {
		return nil, fail("path", name, "not a regular file")
	}
	if info.Size() > model.MaxFileBytes {
		return nil, fail("bounds", name, "file is larger than %d bytes", model.MaxFileBytes)
	}
	file, err := os.Open(full)
	if err != nil {
		return nil, fail("io", name, "cannot open file")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, model.MaxFileBytes+1))
	if err != nil {
		return nil, fail("io", name, "cannot read file")
	}
	if int64(len(data)) > model.MaxFileBytes {
		return nil, fail("bounds", name, "file is larger than %d bytes", model.MaxFileBytes)
	}
	return data, nil
}

func safePath(root, name string) (string, error) {
	if name == "" || strings.Contains(name, "\\") || filepath.IsAbs(name) {
		return "", fail("path", name, "snapshot path must be a relative file inside the input directory")
	}
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fail("path", name, "snapshot path must stay inside the input directory")
	}
	full := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fail("path", name, "snapshot path must stay inside the input directory")
	}
	return full, nil
}
