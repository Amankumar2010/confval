package confval_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Amankumar2010/confval/config"
)

// Fixture-driven tests: every file under testdata/valid must pass the policy,
// and every file under testdata/invalid must fail it. To add a case, drop a
// .yaml/.yml/.json file in the right directory — no code change needed. Point
// these at copies of your real production configs to validate the rule set
// against actual data.

func TestFixturesValid(t *testing.T) {
	runFixtures(t, "testdata/valid", true)
}

func TestFixturesInvalid(t *testing.T) {
	runFixtures(t, "testdata/invalid", false)
}

func runFixtures(t *testing.T, dir string, wantOK bool) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, e := range entries {
		if e.IsDir() || !isConfigFile(e.Name()) {
			continue
		}
		seen++
		path := filepath.Join(dir, e.Name())
		t.Run(e.Name(), func(t *testing.T) {
			c, err := config.Load(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			r := policy().Validate(c)
			if r.OK() != wantOK {
				t.Fatalf("OK() = %v, want %v\n%s", r.OK(), wantOK, r)
			}
		})
	}
	if seen == 0 {
		t.Fatalf("no fixtures found in %s", dir)
	}
}

func isConfigFile(name string) bool {
	switch filepath.Ext(name) {
	case ".yaml", ".yml", ".json":
		return true
	}
	return false
}
