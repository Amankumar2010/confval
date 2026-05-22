package config_test

import (
	"path/filepath"
	"testing"

	"github.com/Amankumar2010/confval/config"
)

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
