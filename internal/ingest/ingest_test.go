package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Harry0052/gridatlas/internal/repo"
)

func TestSampleValidates(t *testing.T) {
	root := mustRoot(t)
	bundle, err := Load(filepath.Join(root, "testdata", "fixtures", "sample"))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Regions) != 3 || len(bundle.Manifest) != 5 {
		t.Fatalf("regions %d manifest %d", len(bundle.Regions), len(bundle.Manifest))
	}
}

func TestInvalidHeaderNamesTheMissingColumn(t *testing.T) {
	root := mustRoot(t)
	_, err := Load(filepath.Join(root, "testdata", "fixtures", "invalid"))
	if err == nil {
		t.Fatal("expected invalid fixture to fail")
	}
	if !strings.Contains(err.Error(), `missing required column "unit"`) {
		t.Fatal(err)
	}
	if _, ok := err.(*Error); !ok {
		t.Fatalf("error type %T", err)
	}
}

func TestRejectsSnapshotPathEscape(t *testing.T) {
	dir := t.TempDir()
	writeFixtureSkeleton(t, dir, "SYN-PATH")
	manifest := strings.Replace(minimalManifest("SYN-PATH", strings.Repeat("ab", 32), "snapshots/installed.csv"), "snapshots/installed.csv", "../secret.csv", 1)
	if err := os.WriteFile(filepath.Join(dir, "manifest.csv"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "inside the input directory") {
		t.Fatal(err)
	}
}

func TestRejectsSymlinkSnapshot(t *testing.T) {
	dir := t.TempDir()
	writeFixtureSkeleton(t, dir, "SYN-LINK")
	outside := filepath.Join(t.TempDir(), "secret.csv")
	body := "record_id,request_id,technology,quantity,unit,basis,row_status\nGEN-1,,Solar,1,MW,nameplate_mw,operating\n"
	if err := os.WriteFile(outside, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "snapshots"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "snapshots", "installed.csv")); err != nil {
		t.Fatal(err)
	}
	manifest := minimalManifest("SYN-LINK", strings.Repeat("ab", 32), "snapshots/installed.csv")
	if err := os.WriteFile(filepath.Join(dir, "manifest.csv"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "symlinks are not allowed") {
		t.Fatal(err)
	}
}

func TestRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	writeFixtureSkeleton(t, dir, "SYN-HASH")
	body := "record_id,request_id,technology,quantity,unit,basis,row_status\nGEN-1,,Solar,1,MW,nameplate_mw,operating\n"
	if err := os.MkdirAll(filepath.Join(dir, "snapshots"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "snapshots", "installed.csv"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := minimalManifest("SYN-HASH", strings.Repeat("ab", 32), "snapshots/installed.csv")
	if err := os.WriteFile(filepath.Join(dir, "manifest.csv"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatal(err)
	}
}

func TestRejectsNonIntegerQuantity(t *testing.T) {
	root := mustRoot(t)
	_, err := Load(filepath.Join(root, "testdata", "fixtures", "invalid"))
	if err == nil {
		t.Fatal("expected failure")
	}
	// The committed invalid fixture fails on the header before the bad quantity.
	// A copied directory with a repaired header must still reject the word "many".
	dir := t.TempDir()
	copyFixture(t, filepath.Join(root, "testdata", "fixtures", "invalid"), dir)
	snapshot, err := os.ReadFile(filepath.Join(dir, "snapshots", "installed.csv"))
	if err != nil {
		t.Fatal(err)
	}
	sum := shaOf(snapshot)
	manifest := minimalManifest("SYN-BAD", sum, "snapshots/installed.csv")
	if err := os.WriteFile(filepath.Join(dir, "manifest.csv"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Load(dir)
	if err == nil || !strings.Contains(err.Error(), "quantity must be a non-negative integer") {
		t.Fatal(err)
	}
}

func writeFixtureSkeleton(t *testing.T, dir, region string) {
	t.Helper()
	files := map[string]string{
		"regions.csv": "region_id,name\n" + region + ",Path Planning Region (synthetic)\n",
		"crosswalk.csv": "source_region,atlas_region,mapping_method,supported\n" +
			region + "," + region + ",explicit_region,true\n",
		"regions.geojson": `{
  "type": "FeatureCollection",
  "features": [{
    "type": "Feature",
    "properties": {"region_id": "` + region + `", "synthetic": true, "boundary_note": "Synthetic rectangle for a path test."},
    "geometry": {"type": "Polygon", "coordinates": [[[0, 0], [1, 0], [1, 1], [0, 1], [0, 0]]]}
  }]
}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func minimalManifest(region, sum, snapshot string) string {
	return "product,source_url,release_date,reporting_period,unit,region,mapping_method,content_sha256,snapshot_file,freshness_threshold_days\n" +
		"installed_nameplate,fixture://test/installed,2026-01-01,2026-01,MW," + region + ",explicit_region," + sum + "," + snapshot + ",90\n"
}

func copyFixture(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func shaOf(body []byte) string {
	return hexSHA(body)
}

func mustRoot(t *testing.T) string {
	t.Helper()
	root, err := repo.Root()
	if err != nil {
		t.Fatal(err)
	}
	return root
}
