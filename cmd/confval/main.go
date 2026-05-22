// Command confval validates configuration files against a set of rules and
// exits non-zero if any file fails, making it a drop-in CI/CD deploy gate.
//
// The rules a project cares about are specific to that project, so this binary
// ships with a small, illustrative rule set (see ruleSet) that you are meant to
// edit, or to copy as a starting point for your own validator binary built on
// the github.com/Amankumar2010/confval library.
//
// Usage:
//
//	confval [flags] <file> [file...]
//
// Flags:
//
//	-quiet   print only failures, not per-file "ok" lines
//
// Exit codes: 0 all files valid, 1 one or more invalid, 2 usage/load error.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Amankumar2010/confval"
	"github.com/Amankumar2010/confval/config"
)

// ruleSet is the validation policy this binary enforces. Replace these rules
// with your own; the library does the rest.
func ruleSet() *confval.Validator {
	return confval.NewValidator(
		confval.Required("service.name", "service.port", "service.environment"),
		confval.IsString("service.name"),
		confval.NonEmpty("service.name"),
		confval.IsNumber("service.port"),
		confval.InRange("service.port", 1, 65535),
		confval.OneOf("service.environment", "development", "staging", "production"),
		confval.Matches("service.name", `^[a-z][a-z0-9-]*$`),
		// A warning, not a failure: replicas above 1 is recommended in prod but
		// not enforced here.
		confval.AsWarning(confval.Predicate("prod_replicas", "service.replicas",
			func(v any) bool {
				n, ok := v.(int)
				return !ok || n >= 1
			}, "should be at least 1")),
	)
}

func main() {
	quiet := flag.Bool("quiet", false, "print only failures, not per-file ok lines")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: confval [-quiet] <file> [file...]\n\n")
		fmt.Fprintf(os.Stderr, "Validates YAML/JSON config files. Exit 0=all valid, 1=invalid, 2=usage error.\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	v := ruleSet()
	failed := false
	for _, path := range files {
		c, err := config.Load(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: load error: %v\n", path, err)
			os.Exit(2)
		}
		report := v.Validate(c)
		if !report.OK() {
			failed = true
		}
		printReport(path, report, *quiet)
	}

	if failed {
		os.Exit(1)
	}
}

func printReport(path string, r *confval.Report, quiet bool) {
	if r.OK() && len(r.Warnings()) == 0 {
		if !quiet {
			fmt.Printf("%s: ok\n", path)
		}
		return
	}
	status := "ok"
	if !r.OK() {
		status = "FAILED"
	}
	fmt.Printf("%s: %s\n", path, status)
	for _, viol := range r.Violations {
		fmt.Printf("  %s\n", viol)
	}
}
