package confval

import (
	"fmt"
	"regexp"

	"github.com/Amankumar2010/confval/config"
)

// This file holds ready-made Rule constructors for the checks configs need
// most. They share two conventions:
//
//   - A field that is absent is reported only by Required; every other rule
//     skips absent paths so optional fields don't generate noise. Compose
//     Required with a type/range rule when a field is both mandatory and
//     constrained.
//   - Numbers are compared after coercion, so a YAML int and a JSON float that
//     represent the same value behave identically.

// Required reports an error for each given path that is absent or explicitly
// null. Use it to assert mandatory keys exist before other rules constrain
// their values.
func Required(paths ...string) Rule {
	return Func("required", func(c *config.Config) []Violation {
		var vs []Violation
		for _, p := range paths {
			v, ok := c.Get(p)
			if !ok || v == nil {
				vs = append(vs, Violation{Path: p, Message: "is required but missing"})
			}
		}
		return vs
	})
}

// fieldRule builds a Rule that runs check against the value at path, but only
// when the path exists and is non-null. check returns a message ("" means ok).
func fieldRule(name, path string, check func(v any) string) Rule {
	return Func(name, func(c *config.Config) []Violation {
		v, ok := c.Get(path)
		if !ok || v == nil {
			return nil
		}
		if msg := check(v); msg != "" {
			return []Violation{{Path: path, Message: msg}}
		}
		return nil
	})
}

// IsString requires the value at path (if present) to be a string.
func IsString(path string) Rule {
	return fieldRule("is_string", path, func(v any) string {
		if _, ok := v.(string); !ok {
			return fmt.Sprintf("must be a string, got %s", typeName(v))
		}
		return ""
	})
}

// IsBool requires the value at path (if present) to be a boolean.
func IsBool(path string) Rule {
	return fieldRule("is_bool", path, func(v any) string {
		if _, ok := v.(bool); !ok {
			return fmt.Sprintf("must be a boolean, got %s", typeName(v))
		}
		return ""
	})
}

// IsNumber requires the value at path (if present) to be numeric.
func IsNumber(path string) Rule {
	return fieldRule("is_number", path, func(v any) string {
		if _, ok := toFloat(v); !ok {
			return fmt.Sprintf("must be a number, got %s", typeName(v))
		}
		return ""
	})
}

// IsList requires the value at path (if present) to be a sequence/array.
func IsList(path string) Rule {
	return fieldRule("is_list", path, func(v any) string {
		if _, ok := v.([]any); !ok {
			return fmt.Sprintf("must be a list, got %s", typeName(v))
		}
		return ""
	})
}

// IsMap requires the value at path (if present) to be a mapping/object.
func IsMap(path string) Rule {
	return fieldRule("is_map", path, func(v any) string {
		if _, ok := v.(map[string]any); !ok {
			return fmt.Sprintf("must be a map, got %s", typeName(v))
		}
		return ""
	})
}

// InRange requires a numeric value at path (if present) to fall within
// [min, max] inclusive.
func InRange(path string, min, max float64) Rule {
	return fieldRule("in_range", path, func(v any) string {
		f, ok := toFloat(v)
		if !ok {
			return fmt.Sprintf("must be a number in [%g, %g], got %s", min, max, typeName(v))
		}
		if f < min || f > max {
			return fmt.Sprintf("must be in [%g, %g], got %g", min, max, f)
		}
		return ""
	})
}

// OneOf requires a string value at path (if present) to be one of allowed.
func OneOf(path string, allowed ...string) Rule {
	set := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		set[a] = true
	}
	return fieldRule("one_of", path, func(v any) string {
		s, ok := v.(string)
		if !ok {
			return fmt.Sprintf("must be one of %v, got %s", allowed, typeName(v))
		}
		if !set[s] {
			return fmt.Sprintf("must be one of %v, got %q", allowed, s)
		}
		return ""
	})
}

// Matches requires a string value at path (if present) to match the regular
// expression. Matches panics at construction time on an invalid pattern,
// surfacing programmer error early rather than at validation time.
func Matches(path, pattern string) Rule {
	re := regexp.MustCompile(pattern)
	return fieldRule("matches", path, func(v any) string {
		s, ok := v.(string)
		if !ok {
			return fmt.Sprintf("must be a string matching %q, got %s", pattern, typeName(v))
		}
		if !re.MatchString(s) {
			return fmt.Sprintf("must match %q, got %q", pattern, s)
		}
		return ""
	})
}

// NonEmpty requires the value at path (if present) to be a non-empty string,
// list, or map.
func NonEmpty(path string) Rule {
	return fieldRule("non_empty", path, func(v any) string {
		switch t := v.(type) {
		case string:
			if t == "" {
				return "must not be empty"
			}
		case []any:
			if len(t) == 0 {
				return "must not be empty"
			}
		case map[string]any:
			if len(t) == 0 {
				return "must not be empty"
			}
		default:
			return fmt.Sprintf("must be a string, list, or map, got %s", typeName(v))
		}
		return ""
	})
}

// Predicate reports an error with message when the value at path is present and
// fails test. test receives the raw decoded value. Use it for one-field checks
// that the typed constructors don't cover.
func Predicate(name, path string, test func(v any) bool, message string) Rule {
	return fieldRule(name, path, func(v any) string {
		if !test(v) {
			return message
		}
		return ""
	})
}

// AsWarning wraps a rule so its violations are reported as warnings rather than
// errors, leaving the config valid (OK) despite them.
func AsWarning(r Rule) Rule {
	return Func(r.Name(), func(c *config.Config) []Violation {
		vs := r.Check(c)
		for i := range vs {
			vs[i].Severity = Warning
		}
		return vs
	})
}

// toFloat coerces JSON/YAML numeric types to float64. JSON decodes numbers as
// float64; YAML may decode them as int or float64.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// typeName gives a config-author-friendly name for a decoded value's type.
func typeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64, float32, int, int64:
		return "number"
	case []any:
		return "list"
	case map[string]any:
		return "map"
	default:
		return fmt.Sprintf("%T", v)
	}
}
