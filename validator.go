// Package confval validates YAML and JSON configuration documents against
// custom business rules written in Go, before they reach production.
//
// The model is deliberately small: a Validator holds an ordered list of Rules,
// each Rule inspects a config.Config and reports zero or more Violations, and
// Validate runs them all and returns a Report. Rules never stop the run early,
// so a single pass surfaces every problem at once.
//
// Common checks (required fields, type and range constraints, enums, regex,
// cross-field invariants) are available as constructors in this package; see
// rules.go. Anything they don't cover is just a func — see RuleFunc.
package confval

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Amankumar2010/confval/config"
)

// Severity classifies a violation. Errors fail validation; warnings are
// reported but leave the config valid.
type Severity int

const (
	Error Severity = iota
	Warning
)

func (s Severity) String() string {
	if s == Warning {
		return "warning"
	}
	return "error"
}

// Violation is a single problem found in a config, anchored to the field path
// that caused it.
type Violation struct {
	// Path is the dotted location of the offending value, e.g.
	// "servers.0.port". Empty for document-level problems.
	Path string
	// Message explains what is wrong, phrased for the person fixing the file.
	Message string
	// Severity is Error unless a rule downgrades it.
	Severity Severity
	// Rule is the name of the rule that produced this violation, for tracing.
	Rule string
}

func (v Violation) String() string {
	loc := v.Path
	if loc == "" {
		loc = "<root>"
	}
	return fmt.Sprintf("%s: %s: %s", v.Severity, loc, v.Message)
}

// Rule inspects a config and appends any problems it finds. A Rule must not
// panic on missing or wrong-typed values; use the helpers in rules.go or guard
// accesses with config.Get's ok return.
type Rule interface {
	// Name identifies the rule in violation output.
	Name() string
	// Check appends Violations for any problems found in c.
	Check(c *config.Config) []Violation
}

// RuleFunc adapts a plain function into a Rule. The name labels violations the
// function reports (and is set on any violation it returns with an empty Rule).
type RuleFunc struct {
	name string
	fn   func(c *config.Config) []Violation
}

// Func builds a Rule from a function. Use it for one-off business logic that
// the built-in constructors don't express.
func Func(name string, fn func(c *config.Config) []Violation) Rule {
	return RuleFunc{name: name, fn: fn}
}

func (r RuleFunc) Name() string { return r.name }

func (r RuleFunc) Check(c *config.Config) []Violation {
	vs := r.fn(c)
	for i := range vs {
		if vs[i].Rule == "" {
			vs[i].Rule = r.name
		}
	}
	return vs
}

// Validator runs an ordered set of rules against configs.
type Validator struct {
	rules []Rule
}

// NewValidator builds a Validator from the given rules.
func NewValidator(rules ...Rule) *Validator {
	return &Validator{rules: append([]Rule(nil), rules...)}
}

// Add appends rules and returns the Validator for chaining.
func (v *Validator) Add(rules ...Rule) *Validator {
	v.rules = append(v.rules, rules...)
	return v
}

// Validate runs every rule against c and returns the combined Report. Rules
// run in registration order; a rule panicking is caught and recorded as an
// error violation so one buggy rule can't abort the whole run.
func (v *Validator) Validate(c *config.Config) *Report {
	r := &Report{Source: c.Source}
	for _, rule := range v.rules {
		r.add(safeCheck(rule, c)...)
	}
	r.sort()
	return r
}

func safeCheck(rule Rule, c *config.Config) (vs []Violation) {
	defer func() {
		if rec := recover(); rec != nil {
			vs = []Violation{{
				Message:  fmt.Sprintf("rule panicked: %v", rec),
				Severity: Error,
				Rule:     rule.Name(),
			}}
		}
	}()
	return rule.Check(c)
}

// Report is the outcome of validating one config.
type Report struct {
	Source     string
	Violations []Violation
}

func (r *Report) add(vs ...Violation) { r.Violations = append(r.Violations, vs...) }

// sort orders violations errors-first, then by path, for stable output.
func (r *Report) sort() {
	sort.SliceStable(r.Violations, func(i, j int) bool {
		a, b := r.Violations[i], r.Violations[j]
		if a.Severity != b.Severity {
			return a.Severity < b.Severity // Error (0) before Warning (1)
		}
		return a.Path < b.Path
	})
}

// OK reports whether the config passed: true when there are no error-severity
// violations. Warnings alone still count as OK.
func (r *Report) OK() bool {
	for _, v := range r.Violations {
		if v.Severity == Error {
			return false
		}
	}
	return true
}

// Errors returns only the error-severity violations.
func (r *Report) Errors() []Violation { return r.filter(Error) }

// Warnings returns only the warning-severity violations.
func (r *Report) Warnings() []Violation { return r.filter(Warning) }

func (r *Report) filter(s Severity) []Violation {
	var out []Violation
	for _, v := range r.Violations {
		if v.Severity == s {
			out = append(out, v)
		}
	}
	return out
}

// String renders the report as a newline-separated, human-readable list.
func (r *Report) String() string {
	if len(r.Violations) == 0 {
		return "ok"
	}
	var b strings.Builder
	for i, v := range r.Violations {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(v.String())
	}
	return b.String()
}
