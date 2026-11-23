// Command gridatlas scores coarse regional grid-data coverage.
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Harry0052/gridatlas/internal/engine"
	"github.com/Harry0052/gridatlas/internal/export"
	"github.com/Harry0052/gridatlas/internal/ingest"
	"github.com/Harry0052/gridatlas/internal/repo"
)

const helpText = `gridatlas — coverage atlas for coarse regional grid-capacity datasets

Usage:
  gridatlas validate --input DIR
  gridatlas run --input DIR --as-of YYYY-MM-DD --out DIR
  gridatlas demo [--as-of YYYY-MM-DD]
  gridatlas serve --dir DIR [--addr 127.0.0.1:8787]
  gridatlas help

The demo reads testdata/fixtures/sample and writes dist/demo.
Run it from the repository root after 'npm run build'.

Exit codes:
  0  success
  2  invalid usage or input
`

func main() {
	os.Exit(Execute(os.Args[1:], os.Stdout, os.Stderr))
}

// Execute runs one command and returns the process exit code.
func Execute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(stdout, helpText)
		return 0
	}
	switch args[0] {
	case "validate":
		return cmdValidate(args[1:], stdout, stderr)
	case "run":
		return cmdRun(args[1:], stdout, stderr)
	case "demo":
		return cmdDemo(args[1:], stdout, stderr)
	case "serve":
		return cmdServe(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		fmt.Fprint(stderr, helpText)
		return 2
	}
}

func cmdValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "fixture directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *input == "" || fs.NArg() != 0 {
		fmt.Fprintln(stderr, "validate requires --input DIR")
		return 2
	}
	bundle, err := ingest.Load(*input)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	fmt.Fprintf(stdout, "valid %s\nregions: %d\nmanifest_rows: %d\n", "gridatlas.input.v1", len(bundle.Regions), len(bundle.Manifest))
	return 0
}

func cmdRun(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "fixture directory")
	asOfText := fs.String("as-of", "", "frozen as-of date YYYY-MM-DD")
	out := fs.String("out", "", "output directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *input == "" || *asOfText == "" || *out == "" || fs.NArg() != 0 {
		fmt.Fprintln(stderr, "run requires --input DIR --as-of YYYY-MM-DD --out DIR")
		return 2
	}
	asOf, err := time.ParseInLocation("2006-01-02", *asOfText, time.UTC)
	if err != nil {
		fmt.Fprintln(stderr, "input invalid (schema): as-of must be YYYY-MM-DD")
		return 2
	}
	return writeRun(*input, asOf, *out, stdout, stderr)
}

func cmdDemo(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("demo", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asOfText := fs.String("as-of", "2026-06-01", "frozen as-of date YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "demo does not take positional arguments")
		return 2
	}
	asOf, err := time.ParseInLocation("2006-01-02", *asOfText, time.UTC)
	if err != nil {
		fmt.Fprintln(stderr, "input invalid (schema): as-of must be YYYY-MM-DD")
		return 2
	}
	root, err := repo.Root()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	input := filepath.Join(root, "testdata", "fixtures", "sample")
	out := filepath.Join(root, "dist", "demo")
	return writeRun(input, asOf, out, stdout, stderr)
}

func writeRun(input string, asOf time.Time, out string, stdout, stderr io.Writer) int {
	outAbs, err := prepareOut(input, out)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	bundle, err := ingest.Load(input)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	atlas, err := engine.Build(bundle, asOf)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	root, err := repo.Root()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	template, err := os.ReadFile(filepath.Join(root, "web", "report.template.html"))
	if err != nil {
		fmt.Fprintln(stderr, "cannot read web/report.template.html; run from the repository root")
		return 2
	}
	script, err := os.ReadFile(filepath.Join(root, "web", "dist", "atlas.js"))
	if err != nil {
		fmt.Fprintln(stderr, "cannot read web/dist/atlas.js; run npm run build from the repository root")
		return 2
	}
	if err := export.WriteDir(outAbs, atlas, bundle.GeoJSON, string(template), script); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	for _, name := range []string{"atlas.json", "summary.md", "regions.geojson", "report.html"} {
		fmt.Fprintf(stdout, "wrote %s\n", filepath.Join(outAbs, name))
	}
	return 0
}

func prepareOut(input, out string) (string, error) {
	absIn, err := filepath.Abs(input)
	if err != nil {
		return "", fmt.Errorf("input invalid (path): cannot resolve input directory")
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return "", fmt.Errorf("input invalid (path): cannot resolve output directory")
	}
	rel, err := filepath.Rel(absIn, absOut)
	if err != nil {
		return "", fmt.Errorf("input invalid (path): cannot compare input and output directories")
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return "", fmt.Errorf("input invalid (path): output directory must not be inside the input directory")
	}
	if err := os.MkdirAll(absOut, 0o755); err != nil {
		return "", fmt.Errorf("input invalid (io): cannot create output directory")
	}
	return absOut, nil
}

func cmdServe(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "", "report directory")
	addr := fs.String("addr", "127.0.0.1:8787", "loopback listen address")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *dir == "" || fs.NArg() != 0 {
		fmt.Fprintln(stderr, "serve requires --dir DIR")
		return 2
	}
	if err := validateAddr(*addr); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	abs, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "input invalid (path): cannot resolve report directory")
		return 2
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		fmt.Fprintln(stderr, "input invalid (path): report directory does not exist")
		return 2
	}
	server := &http.Server{
		Addr:              *addr,
		Handler:           http.FileServer(http.Dir(abs)),
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Fprintf(stdout, "serving %s at http://%s/report.html\n", abs, *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	return 0
}

func validateAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("input invalid (schema): addr must be host:port")
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("input invalid (path): serve binds to a loopback address only")
	}
	return nil
}
