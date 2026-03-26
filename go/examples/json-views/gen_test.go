package jsonviews

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/clarete/langlang/go/extract"
)

func grammarPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// go/examples/json-views/ -> go/ -> repo root
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..",
		"docs", "live", "assets", "examples", "json", "json.peg")
}

func TestGenerateViews(t *testing.T) {
	if os.Getenv("LANGLANG_GENERATE") == "" {
		t.Skip("set LANGLANG_GENERATE=1 to regenerate views")
	}

	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile)

	err := extract.GenerateViews(grammarPath(), "jsonviews", dir, "JSON")
	if err != nil {
		t.Fatal(err)
	}
}
