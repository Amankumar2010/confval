// Package config loads YAML or JSON documents into a uniform, navigable tree.
//
// Both formats decode into the same Go shapes (map[string]any, []any, and
// scalars), so rules written against a Config work identically regardless of
// the source format.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is a parsed configuration document. The zero value is not usable;
// obtain one via Load, LoadBytes, or New.
type Config struct {
	// Source is the file path the config was loaded from, or "" for in-memory
	// documents. It is used to label validation errors.
	Source string
	root   any
}

// New wraps an already-decoded value (map[string]any, []any, or scalar) in a
// Config. Useful for testing rules without touching the filesystem.
func New(root any) *Config {
	return &Config{root: root}
}

// Root returns the top-level decoded value.
func (c *Config) Root() any { return c.root }

// Load reads and parses a file, choosing the parser from its extension:
// .json uses JSON, .yaml/.yml use YAML. Other extensions return an error;
// use LoadBytes to parse with an explicit format.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	format, err := formatFromExt(path)
	if err != nil {
		return nil, err
	}
	c, err := LoadBytes(data, format)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	c.Source = path
	return c, nil
}

// Format identifies a supported serialization.
type Format int

const (
	JSON Format = iota
	YAML
)

func formatFromExt(path string) (Format, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return JSON, nil
	case ".yaml", ".yml":
		return YAML, nil
	default:
		return 0, fmt.Errorf("unsupported file extension %q (want .json, .yaml, or .yml)", filepath.Ext(path))
	}
}

// LoadBytes parses raw bytes using the given format.
func LoadBytes(data []byte, format Format) (*Config, error) {
	var root any
	switch format {
	case JSON:
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("parse JSON: %w", err)
		}
	case YAML:
		if err := yaml.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("parse YAML: %w", err)
		}
		root = normalizeYAML(root)
	default:
		return nil, fmt.Errorf("unknown format %d", format)
	}
	return &Config{root: root}, nil
}

// normalizeYAML converts yaml.v3's map[string]any (with any-typed values that
// may themselves be maps/slices) into the same shape json.Unmarshal produces,
// so downstream rules see one consistent representation. yaml.v3 already keys
// maps as strings for string keys; this walks nested structures to normalize
// numbers and recurse.
func normalizeYAML(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			t[k] = normalizeYAML(val)
		}
		return t
	case map[any]any: // produced for non-string keys; coerce keys to strings
		m := make(map[string]any, len(t))
		for k, val := range t {
			m[fmt.Sprint(k)] = normalizeYAML(val)
		}
		return m
	case []any:
		for i, val := range t {
			t[i] = normalizeYAML(val)
		}
		return t
	default:
		return v
	}
}

// Get navigates a dotted path and returns the value at it. Path segments
// address map keys; numeric segments index into slices. For example,
// "servers.0.port" reads root["servers"][0]["port"].
//
// The returned bool reports whether the path resolved to an existing value.
// A path that resolves to an explicit null returns (nil, true).
func (c *Config) Get(path string) (any, bool) {
	cur := c.root
	if path == "" {
		return cur, true
	}
	for _, seg := range strings.Split(path, ".") {
		switch node := cur.(type) {
		case map[string]any:
			v, ok := node[seg]
			if !ok {
				return nil, false
			}
			cur = v
		case []any:
			i, err := strconv.Atoi(seg)
			if err != nil || i < 0 || i >= len(node) {
				return nil, false
			}
			cur = node[i]
		default:
			return nil, false
		}
	}
	return cur, true
}

// Has reports whether the given path resolves to an existing value.
func (c *Config) Has(path string) bool {
	_, ok := c.Get(path)
	return ok
}
