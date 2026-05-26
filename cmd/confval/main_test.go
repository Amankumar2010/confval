package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write creates a temp file with the given name and contents, returning its
// path. The extension on name drives format detection in config.Load.
func write(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunExitCodes(t *testing.T) {
	valid := "service:\n  name: web-api\n  port: 8080\n  environment: production\n"
	invalid := `{"service":{"name":"Web_API","port":99999,"environment":"prod"}}`

	for _, tc := range []struct {
		name     string
		args     func(t *testing.T) []string
		wantExit int
		wantOut  string // substring expected in stdout
		wantErr  string // substring expected in stderr
	}{
		{
			name:     "valid file passes",
			args:     func(t *testing.T) []string { return []string{write(t, "ok.yaml", valid)} },
			wantExit: 0,
			wantOut:  ": ok",
		},
		{
			name:     "invalid file fails with violations",
			args:     func(t *testing.T) []string { return []string{write(t, "bad.json", invalid)} },
			wantExit: 1,
			wantOut:  "FAILED",
		},
		{
			name:     "no files is a usage error",
			args:     func(t *testing.T) []string { return nil },
			wantExit: 2,
			wantErr:  "usage:",
		},
		{
			name:     "unreadable file is a load error",
			args:     func(t *testing.T) []string { return []string{"does-not-exist.yaml"} },
			wantExit: 2,
			wantErr:  "load error",
		},
		{
			name: "quiet suppresses ok lines",
			args: func(t *testing.T) []string {
				return []string{"-quiet", write(t, "ok.yaml", valid)}
			},
			wantExit: 0,
			wantOut:  "", // nothing printed for a passing file under -quiet
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := run(tc.args(t), &out, &errOut)
			if code != tc.wantExit {
				t.Errorf("exit = %d, want %d (stderr: %s)", code, tc.wantExit, errOut.String())
			}
			if tc.name == "quiet suppresses ok lines" {
				if out.Len() != 0 {
					t.Errorf("quiet mode printed %q, want nothing", out.String())
				}
				return
			}
			if tc.wantOut != "" && !strings.Contains(out.String(), tc.wantOut) {
				t.Errorf("stdout %q missing %q", out.String(), tc.wantOut)
			}
			if tc.wantErr != "" && !strings.Contains(errOut.String(), tc.wantErr) {
				t.Errorf("stderr %q missing %q", errOut.String(), tc.wantErr)
			}
		})
	}
}

func TestRunStopsAtFirstLoadError(t *testing.T) {
	// A bad file mid-list should short-circuit with exit 2.
	good := write(t, "ok.yaml", "service:\n  name: api\n  port: 80\n  environment: staging\n")
	var out, errOut bytes.Buffer
	if code := run([]string{good, "missing.json"}, &out, &errOut); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}
