package export

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Harry0052/gridatlas/internal/model"
)

func TestMarshalEscapesMarkup(t *testing.T) {
	atlas := sampleAtlas()
	atlas.Regions[0].Metrics.InstalledNameplateMW.Provenance.SourceURL = "https://example.test/<script>alert(1)</script>"
	raw, err := Marshal(atlas)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("<script>alert")) {
		t.Fatal("json contained raw markup")
	}
	template := "<script type=\"application/json\" id=\"gridatlas-atlas\">__GRIDATLAS_ATLAS__</script>" +
		"<script type=\"application/json\" id=\"gridatlas-geo\">__GRIDATLAS_GEO__</script>" +
		"<script type=\"application/json\" id=\"gridatlas-summary\">__GRIDATLAS_SUMMARY__</script>" +
		"<script type=\"module\">__GRIDATLAS_SCRIPT__</script>"
	html, err := ReportHTML(template, raw, []byte(`{"type":"FeatureCollection","features":[]}`), []byte("# report"), []byte("const ready = true;"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(html, []byte("<script>alert")) {
		t.Fatal("html contained raw markup")
	}
}

func TestSummaryUsesSeparateMetrics(t *testing.T) {
	text := Summary(sampleAtlas())
	for _, want := range []string{"120 MW", "50 MW", "unobserved", "reported zero", "missing", "incompatible"} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q\n%s", want, text)
		}
	}
	if strings.Contains(text, "70 MW") || strings.Contains(text, "999 MW") {
		t.Fatal(text)
	}
}

func sampleAtlas() *model.Atlas {
	installed := 120
	queued := 50
	zero := 0
	stale := 40
	age := 61
	threshold := 90
	return &model.Atlas{
		SchemaVersion: model.AtlasSchema,
		AsOf:          "2026-06-01",
		Generator:     model.Generator,
		Limitations:   model.Limitations(),
		Regions: []model.RegionResult{
			{
				RegionID: "SYN-NORTH",
				Name:     "North Planning Region (synthetic)",
				Metrics: model.Metrics{
					InstalledNameplateMW: model.Metric{
						Status: model.StatusAvailable, DisplayStatus: model.StatusAvailable,
						ValueMW: &installed, Unit: model.UnitMW, Definition: model.Definition(model.ProductInstalled),
						ReleaseDate: "2026-03-15", Coverage: emptyCoverage(), Trace: []model.TraceItem{},
						Provenance: model.Provenance{AsOf: "2026-06-01", InputSchema: model.InputSchema},
					},
					QueuedRequestedMW: model.Metric{
						Status: model.StatusAvailable, DisplayStatus: model.StatusAvailable,
						ValueMW: &queued, Unit: model.UnitMW, Definition: model.Definition(model.ProductQueued),
						ReleaseDate: "2026-04-01", AgeDays: &age, FreshnessThresholdDays: &threshold,
						Coverage: emptyCoverage(), Trace: []model.TraceItem{},
						Provenance: model.Provenance{AsOf: "2026-06-01", InputSchema: model.InputSchema, SnapshotFile: "snapshots/north-queue.csv"},
					},
					HeadroomMW: model.Headroom{
						Status: model.StatusUnobserved, DisplayStatus: model.StatusUnobserved,
						Unit: model.UnitMW, Definition: model.Definition("headroom"), Reason: model.ReasonNotEstimated,
					},
				},
			},
			{
				RegionID: "SYN-CENTRAL",
				Name:     "Central Planning Region (synthetic)",
				Metrics: model.Metrics{
					InstalledNameplateMW: model.Metric{
						Status: model.StatusAvailable, DisplayStatus: model.StatusAvailable,
						ValueMW: &zero, Zero: true, Unit: model.UnitMW, Definition: model.Definition(model.ProductInstalled),
						Coverage: emptyCoverage(), Trace: []model.TraceItem{},
						Provenance: model.Provenance{AsOf: "2026-06-01", InputSchema: model.InputSchema},
					},
					QueuedRequestedMW: model.Metric{
						Status: model.StatusStale, DisplayStatus: model.StatusStale,
						ValueMW: &stale, Unit: model.UnitMW, Definition: model.Definition(model.ProductQueued),
						Coverage: emptyCoverage(), Trace: []model.TraceItem{},
						Provenance: model.Provenance{AsOf: "2026-06-01", InputSchema: model.InputSchema},
					},
					HeadroomMW: model.Headroom{
						Status: model.StatusUnobserved, DisplayStatus: model.StatusUnobserved,
						Unit: model.UnitMW, Definition: model.Definition("headroom"), Reason: model.ReasonNotEstimated,
					},
				},
			},
			{
				RegionID: "SYN-SOUTH",
				Name:     "South Planning Region (synthetic)",
				Metrics: model.Metrics{
					InstalledNameplateMW: model.Metric{
						Status: model.StatusMissing, DisplayStatus: model.StatusMissing,
						Reason: model.ReasonNoSource, Unit: model.UnitMW, Definition: model.Definition(model.ProductInstalled),
						Coverage: emptyCoverage(), Trace: []model.TraceItem{},
						Provenance: model.Provenance{AsOf: "2026-06-01", InputSchema: model.InputSchema},
					},
					QueuedRequestedMW: model.Metric{
						Status: model.StatusIncompatible, DisplayStatus: model.StatusIncompatible,
						Reason: model.ReasonMislabeledUnit, SourceUnit: "MVA", Unit: model.UnitMW,
						Definition: model.Definition(model.ProductQueued),
						Coverage:   emptyCoverage(), Trace: []model.TraceItem{},
						Provenance: model.Provenance{AsOf: "2026-06-01", InputSchema: model.InputSchema},
					},
					HeadroomMW: model.Headroom{
						Status: model.StatusUnobserved, DisplayStatus: model.StatusUnobserved,
						Unit: model.UnitMW, Definition: model.Definition("headroom"), Reason: model.ReasonNotEstimated,
					},
				},
			},
		},
	}
}

func emptyCoverage() model.Coverage {
	return model.Coverage{IncludedRecordIDs: []string{}}
}

func TestAsOfIsADate(t *testing.T) {
	if sampleAtlas().AsOf != time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02") {
		t.Fatal(sampleAtlas().AsOf)
	}
}
