// Package model holds the versioned input and atlas shapes.
package model

import "time"

const (
	InputSchema = "gridatlas.input.v1"
	AtlasSchema = "gridatlas.atlas.v1"
	Generator   = "gridatlas"

	ProductInstalled = "installed_nameplate"
	ProductQueued    = "queued_requested"

	UnitMW         = "MW"
	BasisNameplate = "nameplate_mw"
	BasisRequested = "requested_mw"
	MethodExplicit = "explicit_region"

	StatusAvailable    = "available"
	StatusStale        = "stale"
	StatusMissing      = "missing"
	StatusIncompatible = "incompatible"
	StatusUnobserved   = "unobserved"
	DisplayUnknown     = "unknown"

	ReasonNoSource             = "no_source"
	ReasonNoDefensibleMapping  = "no_defensible_mapping"
	ReasonUnsupportedJoin      = "unsupported_region_join"
	ReasonMislabeledUnit       = "mislabeled_unit"
	ReasonIncompatibleBasis    = "incompatible_basis"
	ReasonIncompatibleStatus   = "incompatible_row_status"
	ReasonMissingRequestID     = "missing_request_id"
	ReasonReleaseAfterAsOf     = "release_after_as_of"
	ReasonNoIncludedRecords    = "no_included_records"
	ReasonAggregateOutOfBounds = "aggregate_out_of_bounds"
	ReasonNotEstimated         = "not_estimated"

	// MaxAggregateMW refuses a sum that cannot be a coarse regional total in this schema.
	MaxAggregateMW = 10_000_000
	MaxQuantityMW  = 10_000_000
	MaxFileBytes   = 1 << 20
	MaxRows        = 10000
	MaxRegions     = 64
)

// Region is one coarse atlas area. It is not an electrical boundary.
type Region struct {
	ID   string
	Name string
}

// Crosswalk is one versioned source-to-atlas join.
type Crosswalk struct {
	SourceRegion  string
	AtlasRegion   string
	MappingMethod string
	Supported     bool
}

// ManifestRow describes one product snapshot for one region.
type ManifestRow struct {
	Product                string
	SourceURL              string
	ReleaseDate            time.Time
	ReportingPeriod        string
	Unit                   string
	Region                 string
	MappingMethod          string
	ContentSHA256          string
	SnapshotFile           string
	FreshnessThresholdDays int
}

// Measurement is one row inside a snapshot.
type Measurement struct {
	RecordID   string
	RequestID  string
	Technology string
	Quantity   int
	Unit       string
	Basis      string
	RowStatus  string
	Line       int
}

// Bundle is a validated input directory.
type Bundle struct {
	Regions   []Region
	Crosswalk []Crosswalk
	Manifest  []ManifestRow
	Snapshots map[string][]Measurement
	GeoJSON   []byte
}

// Atlas is the versioned machine-readable report.
type Atlas struct {
	SchemaVersion string         `json:"schema_version"`
	AsOf          string         `json:"as_of"`
	Generator     string         `json:"generator"`
	Limitations   []string       `json:"limitations"`
	Regions       []RegionResult `json:"regions"`
}

// RegionResult holds the three metrics for one region. They are never combined.
type RegionResult struct {
	RegionID string  `json:"region_id"`
	Name     string  `json:"name"`
	Metrics  Metrics `json:"metrics"`
}

// Metrics keeps installed, queued, and headroom side by side.
type Metrics struct {
	InstalledNameplateMW Metric   `json:"installed_nameplate_mw"`
	QueuedRequestedMW    Metric   `json:"queued_requested_mw"`
	HeadroomMW           Headroom `json:"headroom_mw"`
}

// Metric is one scored product cell.
type Metric struct {
	Status                 string      `json:"status"`
	DisplayStatus          string      `json:"display_status"`
	ValueMW                *int        `json:"value_mw"`
	Zero                   bool        `json:"zero"`
	Unit                   string      `json:"unit"`
	SourceUnit             string      `json:"source_unit,omitempty"`
	Definition             string      `json:"definition"`
	ReleaseDate            string      `json:"release_date,omitempty"`
	ReportingPeriod        string      `json:"reporting_period,omitempty"`
	FreshnessThresholdDays *int        `json:"freshness_threshold_days,omitempty"`
	AgeDays                *int        `json:"age_days,omitempty"`
	Reason                 string      `json:"reason,omitempty"`
	Coverage               Coverage    `json:"coverage"`
	Provenance             Provenance  `json:"provenance"`
	Trace                  []TraceItem `json:"trace"`
}

// Headroom is never estimated from the other two metrics.
type Headroom struct {
	Status        string `json:"status"`
	DisplayStatus string `json:"display_status"`
	ValueMW       *int   `json:"value_mw"`
	Zero          bool   `json:"zero"`
	Unit          string `json:"unit"`
	Definition    string `json:"definition"`
	Reason        string `json:"reason"`
}

// Coverage counts how the aggregate treated source rows.
type Coverage struct {
	IncludedRecords      int      `json:"included_records"`
	DuplicateRowsIgnored int      `json:"duplicate_rows_ignored"`
	OmittedRows          int      `json:"omitted_rows"`
	WithdrawnRows        int      `json:"withdrawn_rows"`
	AmbiguousRows        int      `json:"ambiguous_rows"`
	AmbiguousRequests    int      `json:"ambiguous_requests"`
	IncludedRecordIDs    []string `json:"included_record_ids"`
}

// Provenance ties a cell to the manifest row that produced it.
type Provenance struct {
	Product         string `json:"product,omitempty"`
	SourceURL       string `json:"source_url,omitempty"`
	ReleaseDate     string `json:"release_date,omitempty"`
	ReportingPeriod string `json:"reporting_period,omitempty"`
	MappingMethod   string `json:"mapping_method,omitempty"`
	ContentSHA256   string `json:"content_sha256,omitempty"`
	SnapshotFile    string `json:"snapshot_file,omitempty"`
	AsOf            string `json:"as_of"`
	InputSchema     string `json:"input_schema"`
}

// TraceItem is one source row and the disposition the scorer gave it.
type TraceItem struct {
	RecordID    string `json:"record_id"`
	RequestID   string `json:"request_id,omitempty"`
	Technology  string `json:"technology,omitempty"`
	QuantityMW  int    `json:"quantity_mw"`
	Unit        string `json:"unit,omitempty"`
	Disposition string `json:"disposition"`
}

// Limitations is the standing note shown with every report.
func Limitations() []string {
	return []string{
		"Regions in this atlas are coarse planning labels. They are not balancing authorities, utility territories, or electrical boundaries.",
		"Installed nameplate MW and queued requested MW are separate published definitions. Neither is available transfer capability.",
		"Headroom is unobserved. GridAtlas does not subtract queued MW from installed MW.",
		"A reported zero stays zero. A missing or unknown cell is not a zero.",
		"A stale cell keeps its original release date and value.",
		"Facility locations, lines, outages, and power-flow results are not part of this report.",
	}
}

// Definition returns the fixed sentence for a product key.
func Definition(product string) string {
	switch product {
	case ProductInstalled:
		return "Sum of operating generator nameplate capacity in MW. Not summer capacity and not available transfer capability."
	case ProductQueued:
		return "Sum of distinct active interconnection requests in requested MW. Duplicate request IDs are counted once. Withdrawn, omitted, and ambiguous requests are excluded. Not operating nameplate and not headroom."
	default:
		return "Unobserved. GridAtlas does not subtract queued MW from installed MW and does not estimate electrical headroom."
	}
}
