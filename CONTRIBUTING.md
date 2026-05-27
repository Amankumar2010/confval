# Contributing to confval

Thanks for your interest — contributions of all sizes are welcome, from typo
fixes to new rules.

## Getting started

```sh
git clone https://github.com/Amankumar2010/confval
cd confval
go test ./...
```

You'll need Go 1.24+ (see `go.mod`). There are no other dependencies beyond
`gopkg.in/yaml.v3`.

## Before you open a PR

CI runs these three checks; please run them locally first so the loop is fast:

```sh
gofmt -l .        # must print nothing — run `gofmt -w .` to fix
go vet ./...
go test -race ./...
```

## What makes a good contribution

- **New built-in rules** belong in `rules.go`. Follow the existing pattern:
  skip absent/null paths (only `Required` reports missing fields), build typed
  rules on the `fieldRule` helper, and write author-friendly messages that name
  the expected value. Add a test in `validator_test.go`.
- **Bug fixes** should come with a test that fails before the fix.
- **Fixtures** are the easiest contribution: drop a `.yaml`/`.yml`/`.json` file
  in `testdata/valid` or `testdata/invalid` and the harness asserts it
  automatically — no code change needed.
- **Keep the public API small.** New exported symbols should earn their place;
  a one-off check is usually better expressed with `confval.Func`.

## Style

- Match the surrounding code — naming, comment density, and idiom.
- Exported identifiers get doc comments that start with the identifier name.
- Prefer table-driven tests.

## Reporting bugs

Open an issue with the config snippet (minimized), the rules you ran, what you
expected, and what you got. A failing config in `testdata/` is the ideal repro.

## License

By contributing, you agree your contributions are licensed under the project's
[MIT License](LICENSE).
