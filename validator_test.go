package confval_test

import (
	"strings"
	"testing"

	"github.com/Amankumar2010/confval"
	"github.com/Amankumar2010/confval/config"
)

// policy mirrors a realistic deploy gate used across the tests.
func policy() *confval.Validator {
	return confval.NewValidator(
		confval.Required("service.name", "service.port", "service.environment"),
		confval.IsString("service.name"),
		confval.Matches("service.name", `^[a-z][a-z0-9-]*$`),
		confval.IsNumber("service.port"),
		confval.InRange("service.port", 1, 65535),
		confval.OneOf("service.environment", "development", "staging", "production"),
	)
}

func TestValidConfigPasses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		format config.Format
		data   string
	}{
		{"yaml", config.YAML, "service:\n  name: web-api\n  port: 8080\n  environment: production\n"},
		{"json", config.JSON, `{"service":{"name":"web-api","port":8080,"environment":"production"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := config.LoadBytes([]byte(tc.data), tc.format)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			r := policy().Validate(c)
			if !r.OK() {
				t.Fatalf("expected OK, got: %s", r)
			}
		})
	}
}

func TestCollectsAllViolations(t *testing.T) {
	// name has bad chars, port out of range, environment not allowed.
	data := "service:\n  name: Web_API\n  port: 99999\n  environment: prod\n"
	c, err := config.LoadBytes([]byte(data), config.YAML)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	r := policy().Validate(c)
	if r.OK() {
		t.Fatal("expected failure")
	}
	if got := len(r.Errors()); got != 3 {
		t.Fatalf("expected 3 errors, got %d:\n%s", got, r)
	}
}

func TestMissingRequiredField(t *testing.T) {
	c, _ := config.LoadBytes([]byte(`{"service":{"name":"api","environment":"staging"}}`), config.JSON)
	r := policy().Validate(c)
	if r.OK() {
		t.Fatal("expected failure for missing port")
	}
	if !strings.Contains(r.String(), "service.port") {
		t.Fatalf("expected port violation, got: %s", r)
	}
}

func TestOptionalFieldSkippedWhenAbsent(t *testing.T) {
	// IsString on an absent optional field must not fail.
	v := confval.NewValidator(confval.IsString("nickname"))
	c := config.New(map[string]any{})
	if r := v.Validate(c); !r.OK() {
		t.Fatalf("absent optional field should pass, got: %s", r)
	}
}

func TestWarningsDoNotFail(t *testing.T) {
	v := confval.NewValidator(
		confval.AsWarning(confval.Required("optional.thing")),
	)
	c := config.New(map[string]any{})
	r := v.Validate(c)
	if !r.OK() {
		t.Fatal("warnings alone must keep config OK")
	}
	if len(r.Warnings()) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(r.Warnings()))
	}
}

func TestPanickingRuleIsContained(t *testing.T) {
	bad := confval.Func("boom", func(*config.Config) []confval.Violation {
		panic("kaboom")
	})
	v := confval.NewValidator(bad)
	r := v.Validate(config.New(nil))
	if r.OK() {
		t.Fatal("panicking rule should produce an error")
	}
	if !strings.Contains(r.String(), "panicked") {
		t.Fatalf("expected panic to be recorded, got: %s", r)
	}
}

func TestErrorsSortBeforeWarnings(t *testing.T) {
	v := confval.NewValidator(
		confval.AsWarning(confval.Required("a")),
		confval.Required("b"),
	)
	r := v.Validate(config.New(map[string]any{}))
	if r.Violations[0].Severity != confval.Error {
		t.Fatalf("errors should sort first, got: %s", r)
	}
}
