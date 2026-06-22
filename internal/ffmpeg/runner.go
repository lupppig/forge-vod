package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Runner executes ffmpeg/ffprobe commands. It is an interface so callers can
// substitute a fake in tests without invoking the real binaries.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// ExecRunner runs commands using os/exec against the real binaries on PATH.
type ExecRunner struct {
	FFmpegPath  string
	FFprobePath string
}

// NewExecRunner returns an ExecRunner defaulting to "ffmpeg"/"ffprobe" on PATH.
func NewExecRunner() *ExecRunner {
	return &ExecRunner{FFmpegPath: "ffmpeg", FFprobePath: "ffprobe"}
}

// Run executes name with args, returning combined stdout. On failure the error
// includes stderr to make ffmpeg diagnostics visible.
func (r *ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s failed: %w: %s", name, err, stderr.String())
	}
	return stdout.Bytes(), nil
}
