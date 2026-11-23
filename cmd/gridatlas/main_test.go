package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Harry0052/gridatlas/internal/repo"
)

func TestHelpAndUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit %d", code)
	}
	if !strings.Contains(stdout.String(), "gridatlas validate") {
		t.Fatal(stdout.String())
	}
	stdout.Reset()
	if code := Execute([]string{"nope"}, &stdout, &stderr); code != 2 {
		t.Fatalf("unknown exit %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatal(stderr.String())
	}
}

func TestValidateSampleAndRejectInvalid(t *testing.T) {
	root := mustRoot(t)
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"validate", "--input", filepath.Join(root, "testdata", "fixtures", "sample")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("validate exit %d %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "valid gridatlas.input.v1") || !strings.Contains(stdout.String(), "regions: 3") {
		t.Fatal(stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"run", "--input", filepath.Join(root, "testdata", "fixtures", "invalid"), "--as-of", "2026-06-01", "--out", filepath.Join(root, "dist", "invalid-out")}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("invalid exit %d stdout %s stderr %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), `missing required column "unit"`) {
		t.Fatal(stderr.String())
	}
}

func TestServeRejectsPublicBind(t *testing.T) {
	root := mustRoot(t)
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"serve", "--dir", root, "--addr", "0.0.0.0:8787"}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "loopback") {
		t.Fatalf("exit %d %s", code, stderr.String())
	}
}

func mustRoot(t *testing.T) string {
	t.Helper()
	root, err := repo.Root()
	if err != nil {
		t.Fatal(err)
	}
	return root
}
