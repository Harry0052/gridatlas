package engine

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Harry0052/gridatlas/internal/ingest"
	"github.com/Harry0052/gridatlas/internal/model"
	"github.com/Harry0052/gridatlas/internal/repo"
)

func TestOracleFixtures(t *testing.T) {
	root := mustRoot(t)
	cases := []struct {
		fixture string
		oracle  string
	}{
		{"testdata/fixtures/sample", "testdata/oracle/sample.txt"},
		{"testdata/fixtures/boundary", "testdata/oracle/boundary.txt"},
		{"testdata/fixtures/unknown", "testdata/oracle/unknown.txt"},
		{"testdata/fixtures/unsupported", "testdata/oracle/unsupported.txt"},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			bundle, err := ingest.Load(filepath.Join(root, tc.fixture))
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			sections := parseOracle(t, filepath.Join(root, tc.oracle))
			if len(sections) == 0 {
				t.Fatal("oracle has no as_of section")
			}
			for _, section := range sections {
				asOf, err := time.ParseInLocation("2006-01-02", section.asOf, time.UTC)
				if err != nil {
					t.Fatal(err)
				}
				atlas, err := Build(bundle, asOf)
				if err != nil {
					t.Fatalf("build %s: %v", section.asOf, err)
				}
				if atlas.SchemaVersion != model.AtlasSchema {
					t.Fatalf("schema %s", atlas.SchemaVersion)
				}
				if atlas.AsOf != section.asOf {
					t.Fatalf("as of %s", atlas.AsOf)
				}
				got := map[string][4]string{}
				for _, region := range atlas.Regions {
					for _, name := range []string{"installed_nameplate_mw", "queued_requested_mw", "headroom_mw"} {
						got[region.RegionID+" "+name] = viewOf(region, name)
					}
					if region.Metrics.InstalledNameplateMW.Reason == model.ReasonNoDefensibleMapping &&
						region.Metrics.InstalledNameplateMW.DisplayStatus != model.DisplayUnknown {
						t.Fatalf("%s display status %s", region.RegionID, region.Metrics.InstalledNameplateMW.DisplayStatus)
					}
					head := region.Metrics.HeadroomMW
					if head.Status != model.StatusUnobserved || head.ValueMW != nil || head.Reason != model.ReasonNotEstimated {
						t.Fatalf("%s headroom = %+v", region.RegionID, head)
					}
				}
				if len(got) != len(section.rows) {
					t.Fatalf("as of %s: atlas has %d cells, oracle has %d", section.asOf, len(got), len(section.rows))
				}
				for key, want := range section.rows {
					have, ok := got[key]
					if !ok {
						t.Fatalf("missing cell %s", key)
					}
					if have != want {
						t.Fatalf("%s at %s: got %q want %q", key, section.asOf, have, want)
					}
				}
			}
		})
	}
}

func TestSampleDoesNotPublishRejectedQuantities(t *testing.T) {
	root := mustRoot(t)
	bundle, err := ingest.Load(filepath.Join(root, "testdata", "fixtures", "sample"))
	if err != nil {
		t.Fatal(err)
	}
	atlas, err := Build(bundle, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for _, region := range atlas.Regions {
		for _, value := range []*int{
			region.Metrics.InstalledNameplateMW.ValueMW,
			region.Metrics.QueuedRequestedMW.ValueMW,
			region.Metrics.HeadroomMW.ValueMW,
		} {
			if value != nil && (*value == 70 || *value == 999 || *value == 15 || *value == 25 || *value == 80) {
				t.Fatalf("published rejected quantity %d for %s", *value, region.RegionID)
			}
		}
		installed := region.Metrics.InstalledNameplateMW.ValueMW
		queued := region.Metrics.QueuedRequestedMW.ValueMW
		if installed != nil && queued != nil && region.Metrics.HeadroomMW.ValueMW != nil {
			t.Fatal("headroom was estimated")
		}
	}
	central := findRegion(t, atlas, "SYN-CENTRAL")
	if !central.Metrics.InstalledNameplateMW.Zero || central.Metrics.InstalledNameplateMW.Status != model.StatusAvailable {
		t.Fatalf("central zero = %+v", central.Metrics.InstalledNameplateMW)
	}
	south := findRegion(t, atlas, "SYN-SOUTH")
	if south.Metrics.InstalledNameplateMW.ValueMW != nil || south.Metrics.InstalledNameplateMW.Zero {
		t.Fatal("missing installed was stored as a number")
	}
	if len(south.Metrics.QueuedRequestedMW.Trace) != 1 || south.Metrics.QueuedRequestedMW.Trace[0].Unit != "MVA" {
		t.Fatalf("rejected unit was relabeled: %+v", south.Metrics.QueuedRequestedMW.Trace)
	}
	north := findRegion(t, atlas, "SYN-NORTH")
	if north.Metrics.QueuedRequestedMW.Coverage.DuplicateRowsIgnored != 1 ||
		north.Metrics.QueuedRequestedMW.Coverage.AmbiguousRequests != 1 ||
		north.Metrics.QueuedRequestedMW.Coverage.WithdrawnRows != 1 ||
		north.Metrics.QueuedRequestedMW.Coverage.OmittedRows != 1 {
		t.Fatalf("queue coverage = %+v", north.Metrics.QueuedRequestedMW.Coverage)
	}
}

func TestRepeatedRequestWithTheSameMWCountsOnce(t *testing.T) {
	release := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	bundle := &model.Bundle{
		Regions: []model.Region{{ID: "SYN-NORTH", Name: "North"}},
		Crosswalk: []model.Crosswalk{{
			SourceRegion: "SYN-NORTH", AtlasRegion: "SYN-NORTH", MappingMethod: model.MethodExplicit, Supported: true,
		}},
		Manifest: []model.ManifestRow{{
			Product: model.ProductQueued, SourceURL: "fixture://unit/queue", ReleaseDate: release,
			ReportingPeriod: "2026-05", Unit: model.UnitMW, Region: "SYN-NORTH",
			MappingMethod: model.MethodExplicit, SnapshotFile: "q.csv", FreshnessThresholdDays: 90,
		}},
		Snapshots: map[string][]model.Measurement{
			"q.csv": {
				{RecordID: "A", RequestID: "Q-1", Technology: "Solar", Quantity: 40, Unit: "MW", Basis: model.BasisRequested, RowStatus: "active", Line: 2},
				{RecordID: "B", RequestID: "Q-1", Technology: "Wind", Quantity: 40, Unit: "MW", Basis: model.BasisRequested, RowStatus: "active", Line: 3},
			},
		},
	}
	atlas, err := Build(bundle, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	got := atlas.Regions[0].Metrics.QueuedRequestedMW
	if got.ValueMW == nil || *got.ValueMW != 40 || got.Coverage.DuplicateRowsIgnored != 1 {
		t.Fatalf("same-MW multi-technology request = %+v", got)
	}
}

func TestMislabeledKilowattsAreNotConverted(t *testing.T) {
	release := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	bundle := &model.Bundle{
		Regions: []model.Region{{ID: "SYN-NORTH", Name: "North"}},
		Crosswalk: []model.Crosswalk{{
			SourceRegion: "SYN-NORTH", AtlasRegion: "SYN-NORTH", MappingMethod: model.MethodExplicit, Supported: true,
		}},
		Manifest: []model.ManifestRow{{
			Product: model.ProductInstalled, SourceURL: "fixture://unit/installed", ReleaseDate: release,
			ReportingPeriod: "2026-05", Unit: "kW", Region: "SYN-NORTH",
			MappingMethod: model.MethodExplicit, SnapshotFile: "i.csv", FreshnessThresholdDays: 90,
		}},
		Snapshots: map[string][]model.Measurement{
			"i.csv": {
				{RecordID: "G", Technology: "Solar", Quantity: 1000, Unit: "kW", Basis: model.BasisNameplate, RowStatus: "operating", Line: 2},
			},
		},
	}
	atlas, err := Build(bundle, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	got := atlas.Regions[0].Metrics.InstalledNameplateMW
	if got.Status != model.StatusIncompatible || got.Reason != model.ReasonMislabeledUnit || got.ValueMW != nil {
		t.Fatalf("kW cell = %+v", got)
	}
}

func TestDemoBudget(t *testing.T) {
	root := mustRoot(t)
	bundle, err := ingest.Load(filepath.Join(root, "testdata", "fixtures", "sample"))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := Build(bundle, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed > time.Second {
		t.Fatalf("sample score took %s, budget is 1s", elapsed)
	}
}

func findRegion(t *testing.T, atlas *model.Atlas, id string) model.RegionResult {
	t.Helper()
	for _, region := range atlas.Regions {
		if region.RegionID == id {
			return region
		}
	}
	t.Fatalf("missing region %s", id)
	return model.RegionResult{}
}

func viewOf(region model.RegionResult, name string) [4]string {
	switch name {
	case "headroom_mw":
		item := region.Metrics.HeadroomMW
		return [4]string{dashPtr(item.ValueMW), item.Status, "-", item.Reason}
	case "queued_requested_mw":
		return viewMetric(region.Metrics.QueuedRequestedMW)
	default:
		return viewMetric(region.Metrics.InstalledNameplateMW)
	}
}

func viewMetric(metric model.Metric) [4]string {
	release := metric.ReleaseDate
	if release == "" {
		release = "-"
	}
	reason := metric.Reason
	if reason == "" {
		reason = "-"
	}
	return [4]string{dashPtr(metric.ValueMW), metric.Status, release, reason}
}

func dashPtr(value *int) string {
	if value == nil {
		return "-"
	}
	return strconv.Itoa(*value)
}

type oracleSection struct {
	asOf string
	rows map[string][4]string
}

func parseOracle(t *testing.T, path string) []oracleSection {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var sections []oracleSection
	var current *oracleSection
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "as_of" {
			sections = append(sections, oracleSection{asOf: fields[1], rows: map[string][4]string{}})
			current = &sections[len(sections)-1]
			continue
		}
		if current == nil || len(fields) != 6 {
			t.Fatalf("bad oracle line %q", line)
		}
		current.rows[fields[0]+" "+fields[1]] = [4]string{fields[2], fields[3], fields[4], fields[5]}
	}
	return sections
}

func mustRoot(t *testing.T) string {
	t.Helper()
	root, err := repo.Root()
	if err != nil {
		wd, _ := os.Getwd()
		for {
			if _, statErr := os.Stat(filepath.Join(wd, "go.mod")); statErr == nil {
				return wd
			}
			parent := filepath.Dir(wd)
			if parent == wd {
				t.Fatal(err)
			}
			wd = parent
		}
	}
	return root
}
