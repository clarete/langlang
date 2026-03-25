package python

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	langlang "github.com/clarete/langlang/go"
	"github.com/clarete/langlang/go/corpus"

	"github.com/stretchr/testify/require"
)

var corpusSkipDirs = map[string]bool{
	".git":          true,
	"__pycache__":   true,
	".tox":          true,
	".venv":         true,
	"venv":          true,
	"node_modules":  true,
	".mypy_cache":   true,
	".pytest_cache": true,
	"dist":          true,
	"build":         true,
	"*.egg-info":    true,
}

// TestPythonDifferentialCorpus walks a real-world Python codebase and
// parses every .py file with the Python grammar.  Since the corpus
// comes from a working project, every file is assumed to be valid
// Python; any parse failure is a grammar gap.
//
//	go test -run TestPythonDifferentialCorpus -v -timeout 30m \
//		 ./tests/python/ -args -corpus=~/path/to/python/repo
func TestPythonDifferentialCorpus(t *testing.T) {
	matcher, err := langlang.MatcherFromFilePathWithConfig(grammarPath, pythonConfig())
	require.NoError(t, err)
	corpus.RunFromFlag(t, corpus.Config{
		Extensions:     []string{".py"},
		Matcher:        matcher,
		SkipDirs:       corpusSkipDirs,
		FailThreshold:  10.0,
		PerFileTimeout: 10 * time.Second,
		LangName:       "Python",
	})
}

// CorpusTier orders repos from smallest to largest for gradual corpus rollout.
// Tier 1: in-repo testdata only. Tier 2+: cloned repos (need -corpus_cache and AUTO_CORPUS or -corpus).
var corpusTiers = [][]corpus.Repo{
	{}, // tier 0 unused
	{}, // tier 1 = testdata only (no repos)
	{{Name: "requests", URL: "https://github.com/psf/requests.git", Branch: ""}},
	{
		{Name: "flask", URL: "https://github.com/pallets/flask.git", Branch: ""},
		{Name: "black", URL: "https://github.com/psf/black.git", Branch: ""},
		{Name: "mypy", URL: "https://github.com/python/mypy.git", Branch: ""},
	},
	{
		{Name: "django", URL: "https://github.com/django/django.git", Branch: "main"},
		{Name: "fastapi", URL: "https://github.com/tiangolo/fastapi.git", Branch: ""},
	},
	// tier 5: largest (e.g. cpython) — add when tier 4 is stable
	// {{Name: "cpython", URL: "https://github.com/python/cpython.git", Branch: "main"}},
}

var defaultCorpusRepos = []corpus.Repo{
	{Name: "flask", URL: "https://github.com/pallets/flask.git", Branch: ""},
	{Name: "requests", URL: "https://github.com/psf/requests.git", Branch: ""},
	{Name: "django", URL: "https://github.com/django/django.git", Branch: "main"},
	{Name: "fastapi", URL: "https://github.com/tiangolo/fastapi.git", Branch: ""},
	{Name: "black", URL: "https://github.com/psf/black.git", Branch: ""},
	{Name: "mypy", URL: "https://github.com/python/mypy.git", Branch: ""},
}

// CorpusTierFlag selects which tier to run (1 = testdata only, 2+ = that tier's repos).
var CorpusTierFlag = flag.Int("corpus_tier", 0, "run corpus tier only (1=testdata, 2+=repos); 0=all")

// CorpusRepoFlag runs a single repo in TestPythonAutoCorpus (e.g. -corpus_repo=flask). Empty = all repos.
var CorpusRepoFlag = flag.String("corpus_repo", "", "run only this repo in TestPythonAutoCorpus (flask, requests, django, fastapi, black, mypy)")

// TestPythonCorpusTier runs a single corpus tier. Tier 1: testdata + corpus_repros (no clone).
// Tier 2+: clone that tier's repos (need AUTO_CORPUS=true and -corpus_cache=dir).
// Run with: go test -run TestPythonCorpusTier -v ./tests/python/ -args -corpus_tier=1
func TestPythonCorpusTier(t *testing.T) {
	flag.Parse()
	tier := *CorpusTierFlag
	if tier <= 0 || tier >= len(corpusTiers) {
		t.Skipf("run with -corpus_tier=1..%d", len(corpusTiers)-1)
		return
	}
	matcher, err := langlang.MatcherFromFilePathWithConfig(grammarPath, pythonConfig())
	require.NoError(t, err)
	if tier == 1 {
		corpus.RunTestFiles(t, matcher, testdataDir, ".py")
		reprosDir := filepath.Join(testdataDir, "corpus_repros")
		if _, err := os.Stat(reprosDir); err == nil {
			corpus.RunTestFiles(t, matcher, reprosDir, ".py")
		}
		return
	}
	if os.Getenv("AUTO_CORPUS") != "true" {
		t.Skip("set AUTO_CORPUS=true and -corpus_cache=dir for tier >= 2")
		return
	}
	baseDir, useCache := corpus.CorpusCacheDirExpanded()
	if !useCache {
		t.Skip("set -corpus_cache=dir for tier >= 2")
		return
	}
	for _, repo := range corpusTiers[tier] {
		t.Run(repo.Name, func(t *testing.T) {
			dest := filepath.Join(baseDir, repo.Name)
			corpus.CloneRepoIfNeeded(t, repo, dest)
			corpus.Run(t, corpus.Config{
				Dir:            dest,
				Extensions:     []string{".py"},
				Matcher:        matcher,
				SkipDirs:       corpusSkipDirs,
				FailThreshold:  0,
				PerFileTimeout: 10 * time.Second,
				LangName:       "Python",
			})
		})
	}
}

// TestPythonPathBCorpusTier runs the Path B pipeline (tokenizer + rewrite parse) on the same
// corpus as TestPythonCorpusTier. Use -corpus_tier=1 for testdata+repros, or tier 2+ with
// AUTO_CORPUS=true and -corpus_cache=dir. Path B success = tokenize consumes all + parse rewrite succeeds.
//
//	go test -run TestPythonPathBCorpusTier -v ./tests/python/ -args -corpus_tier=1
func TestPythonPathBCorpusTier(t *testing.T) {
	flag.Parse()
	tier := *CorpusTierFlag
	if tier <= 0 || tier >= len(corpusTiers) {
		t.Skipf("run with -corpus_tier=1..%d", len(corpusTiers)-1)
		return
	}
	matcher := newPathBMatcher(t)
	if tier == 1 {
		// Path B tier 1: per-file timeout; pathBMatcher serializes Match via mutex
		cfg := corpus.Config{
			Dir:            testdataDir,
			Extensions:     []string{".py"},
			Matcher:        matcher,
			FailThreshold:  0,
			PerFileTimeout: 10 * time.Second,
			LangName:       "Python (Path B)",
		}
		corpus.Run(t, cfg)
		reprosDir := filepath.Join(testdataDir, "corpus_repros")
		if _, err := os.Stat(reprosDir); err == nil {
			cfg.Dir = reprosDir
			corpus.Run(t, cfg)
		}
		return
	}
	if os.Getenv("AUTO_CORPUS") != "true" {
		t.Skip("set AUTO_CORPUS=true and -corpus_cache=dir for tier >= 2")
		return
	}
	baseDir, useCache := corpus.CorpusCacheDirExpanded()
	if !useCache {
		t.Skip("set -corpus_cache=dir for tier >= 2")
		return
	}
	for _, repo := range corpusTiers[tier] {
		t.Run(repo.Name, func(t *testing.T) {
			dest := filepath.Join(baseDir, repo.Name)
			corpus.CloneRepoIfNeeded(t, repo, dest)
			corpus.Run(t, corpus.Config{
				Dir:            dest,
				Extensions:     []string{".py"},
				Matcher:        matcher,
				SkipDirs:       corpusSkipDirs,
				FailThreshold:  0,
				PerFileTimeout: 5 * time.Second,
				LangName:       "Python (Path B)",
			})
		})
	}
}

// TestPythonAutoCorpus clones well-known Python repos and parses
// every .py file.  Needs network.  Use -corpus_cache=dir to cache
// cloned repos and avoid re-downloading.
//
//	AUTO_CORPUS=true go test -run TestPythonAutoCorpus -v -timeout 60m \
//		./tests/python/
//	AUTO_CORPUS=true go test -run TestPythonAutoCorpus -v -timeout 60m \
//		./tests/python/ -args -corpus_cache=~/cache/python-corpus
//
// Run one repo at a time (e.g. to avoid long runs):
//
//	AUTO_CORPUS=true go test -run TestPythonAutoCorpus -v ./tests/python/ -args -corpus_cache=../.corpus -corpus_repo=flask
func TestPythonAutoCorpus(t *testing.T) {
	if os.Getenv("AUTO_CORPUS") != "true" {
		t.Skip()
	}
	flag.Parse()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	baseDir, useCache := corpus.CorpusCacheDirExpanded()
	if !useCache {
		var err error
		baseDir, err = os.MkdirTemp("", "langlang-python-corpus-*")
		if err != nil {
			t.Fatalf("cannot create temp dir: %v", err)
		}
		defer os.RemoveAll(baseDir)
	} else {
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			t.Fatalf("cannot create cache dir %s: %v", baseDir, err)
		}
	}

	onlyRepo := *CorpusRepoFlag
	if onlyRepo != "" {
		found := false
		for _, r := range defaultCorpusRepos {
			if r.Name == onlyRepo {
				found = true
				break
			}
		}
		if !found {
			t.Skipf("unknown -corpus_repo=%q; use one of: flask, requests, django, fastapi, black, mypy", onlyRepo)
		}
	}
	for _, repo := range defaultCorpusRepos {
		if onlyRepo != "" && repo.Name != onlyRepo {
			continue
		}
		t.Run(repo.Name, func(t *testing.T) {
			dest := filepath.Join(baseDir, repo.Name)
			corpus.CloneRepoIfNeeded(t, repo, dest)
			matcher, err := langlang.MatcherFromFilePathWithConfig(grammarPath, pythonConfig())
			require.NoError(t, err)
			corpus.Run(t, corpus.Config{
				Dir:            dest,
				Extensions:     []string{".py"},
				Matcher:        matcher,
				SkipDirs:       corpusSkipDirs,
				FailThreshold:  0,
				PerFileTimeout: 10 * time.Second,
				LangName:       "Python",
			})
		})
	}
}
