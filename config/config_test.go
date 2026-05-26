package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Amankumar2010/confval/config"
)

func writeTemp(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadByExtension(t *testing.T) {
	for _, tc := range []struct{ name, body, path string }{
		{"yaml", "a: 1\n", "c.yaml"},
		{"yml", "a: 1\n", "c.yml"},
		{"json", `{"a":1}`, "c.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := config.Load(writeTemp(t, tc.path, tc.body))
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if v, ok := c.Get("a"); !ok || toF(v) != 1 {
				t.Fatalf("a = %v, ok=%v", v, ok)
			}
			if c.Source == "" {
				t.Error("Source should be set after Load")
			}
		})
	}
}

func toF(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case float64:
		return n
	}
	return -1
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := config.Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadMalformed(t *testing.T) {
	if _, err := config.Load(writeTemp(t, "bad.json", `{"a":`)); err == nil {
		t.Fatal("expected parse error for malformed JSON")
	}
	if _, err := config.Load(writeTemp(t, "bad.yaml", "a:\n  - b\n c\n")); err == nil {
		t.Fatal("expected parse error for malformed YAML")
	}
}

func TestHas(t *testing.T) {
	c, _ := config.LoadBytes([]byte(`{"a":{"b":1}}`), config.JSON)
	if !c.Has("a.b") {
		t.Error("a.b should exist")
	}
	if c.Has("a.z") {
		t.Error("a.z should not exist")
	}
}

func TestNonStringYAMLKeysCoerced(t *testing.T) {
	// A YAML mapping with an integer key should be addressable as a string.
	c, err := config.LoadBytes([]byte("1: one\ntrue: yes-key\n"), config.YAML)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := c.Get("1"); !ok || v != "one" {
		t.Fatalf("key 1 = %v, ok=%v", v, ok)
	}
}

func TestEmptyPathReturnsRoot(t *testing.T) {
	c := config.New(map[string]any{"x": 1})
	if v, ok := c.Get(""); !ok || v == nil {
		t.Fatalf("empty path should return root, got %v ok=%v", v, ok)
	}
}

func TestUnknownFormat(t *testing.T) {
	if _, err := config.LoadBytes([]byte("{}"), config.Format(99)); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestGetNavigatesMapsAndSlices(t *testing.T) {
	c, err := config.LoadBytes([]byte(`{"servers":[{"port":80},{"port":443}]}`), config.JSON)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := c.Get("servers.1.port")
	if !ok {
		t.Fatal("expected to resolve servers.1.port")
	}
	if v.(float64) != 443 {
		t.Fatalf("got %v", v)
	}
}

func TestGetMissingPaths(t *testing.T) {
	c, _ := config.LoadBytes([]byte(`{"a":{"b":1}}`), config.JSON)
	for _, p := range []string{"a.c", "x", "a.b.c", "a.5"} {
		if _, ok := c.Get(p); ok {
			t.Errorf("expected %q to be absent", p)
		}
	}
}

func TestExplicitNullResolves(t *testing.T) {
	c, _ := config.LoadBytes([]byte(`{"a":null}`), config.JSON)
	v, ok := c.Get("a")
	if !ok || v != nil {
		t.Fatalf("explicit null should resolve to (nil, true), got (%v, %v)", v, ok)
	}
}

func TestYAMLAndJSONDecodeAlike(t *testing.T) {
	y, _ := config.LoadBytes([]byte("a:\n  b: hello\n"), config.YAML)
	j, _ := config.LoadBytes([]byte(`{"a":{"b":"hello"}}`), config.JSON)
	yv, _ := y.Get("a.b")
	jv, _ := j.Get("a.b")
	if yv != jv {
		t.Fatalf("yaml %v != json %v", yv, jv)
	}
}

func TestLoadUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	if _, err := config.Load(filepath.Join(dir, "x.toml")); err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}
