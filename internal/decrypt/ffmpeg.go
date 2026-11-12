// Package decrypt removes the DRM from a downloaded Audible payload.
//
// The package exposes a Decryptor interface. The shipped implementation
// delegates to ffmpeg's native aax demuxer, which understands the AAXC
// container and accepts the account-scoped key and IV directly:
//
//	ffmpeg -audible_key <hex> -audible_iv <hex> -i in.aaxc -c copy out.m4b
//
// A pure-Go implementation can be added later behind the same interface.
//
// Layer L3 (domain).
package decrypt

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Decryptor removes the DRM from an encrypted payload.
type Decryptor interface {
	// Decrypt converts the encrypted input into a plain output file.
	//
	// key and iv are hex strings. durationSeconds is the runtime of the
	// title, used to turn ffmpeg's position into a ratio. onProgress
	// receives a ratio between 0 and 1 as the conversion proceeds.
	Decrypt(ctx context.Context, input, output, key, iv string, durationSeconds float64, onProgress func(float64)) error

	// Channels reports the number of audio channels in a file.
	Channels(ctx context.Context, path string) (int, error)
}

// FFmpeg is a Decryptor that shells out to ffmpeg.
type FFmpeg struct {
	// Path is the location of the ffmpeg executable. When empty, ffmpeg is
	// looked up on PATH.
	Path string
	// FFprobePath is the location of ffprobe. When empty, ffprobe is
	// looked up on PATH.
	FFprobePath string
}

// NewFFmpeg creates a Decryptor that uses ffmpeg.
func NewFFmpeg() *FFmpeg { return &FFmpeg{} }

// Resolve returns the path of the ffmpeg executable.
func (f *FFmpeg) Resolve() string {
	if f.Path != "" {
		return f.Path
	}
	if p := os.Getenv("UNSEAL_FFMPEG"); p != "" {
		return p
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	return "ffmpeg"
}

// ResolveFFprobe returns the path of the ffprobe executable.
func (f *FFmpeg) ResolveFFprobe() string {
	if f.FFprobePath != "" {
		return f.FFprobePath
	}
	if p := os.Getenv("UNSEAL_FFPROBE"); p != "" {
		return p
	}
	if p, err := exec.LookPath("ffprobe"); err == nil {
		return p
	}
	return "ffprobe"
}

// Decrypt converts the encrypted input into a plain output file.
func (f *FFmpeg) Decrypt(ctx context.Context, input, output, key, iv string, durationSeconds float64, onProgress func(float64)) error {
	ffmpeg := f.Resolve()

	args := []string{
		"-y",
		"-audible_key", key,
		"-audible_iv", iv,
		"-i", input,
		"-c", "copy",
		"-progress", "pipe:1",
		output,
	}

	cmd := exec.CommandContext(ctx, ffmpeg, args...)
	// Hide the console window on Windows.
	hideConsole(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("decrypt: cannot open the progress pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("decrypt: cannot open the error pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("decrypt: cannot start ffmpeg: %w", err)
	}

	// Drain stderr in the background so ffmpeg does not block.
	go func() {
		_, _ = io.Copy(io.Discard, stderr)
	}()

	// Parse the progress output. ffmpeg writes "out_time_ms=<us>" lines
	// as it converts, followed by "progress=end" when it is done.
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_ms=") {
			ms, err := strconv.ParseInt(strings.TrimPrefix(line, "out_time_ms="), 10, 64)
			if err == nil && ms > 0 && onProgress != nil {
				seconds := float64(ms) / 1e6
				if durationSeconds > 0 {
					ratio := seconds / durationSeconds
					if ratio > 1 {
						ratio = 1
					}
					onProgress(ratio)
				}
			}
		}
		if strings.HasPrefix(line, "progress=") {
			if strings.TrimPrefix(line, "progress=") == "end" {
				break
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("decrypt: ffmpeg failed: %w", err)
	}

	if onProgress != nil {
		onProgress(1.0)
	}
	return nil
}

// Channels reports the number of audio channels in a file.
func (f *FFmpeg) Channels(ctx context.Context, path string) (int, error) {
	ffprobe := f.ResolveFFprobe()

	cmd := exec.CommandContext(ctx, ffprobe,
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=channels",
		"-of", "csv=p=0",
		path,
	)
	hideConsole(cmd)

	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("decrypt: ffprobe failed: %w", err)
	}

	s := strings.TrimSpace(string(out))
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("decrypt: cannot parse the channel count %q: %w", s, err)
	}
	return n, nil
}

// ErrFFmpegNotFound is returned when ffmpeg cannot be found.
var ErrFFmpegNotFound = fmt.Errorf("decrypt: ffmpeg is not installed")
