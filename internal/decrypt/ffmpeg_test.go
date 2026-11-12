package decrypt_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ngbcoder/Unseal/internal/decrypt"
)

func TestResolveFindsFFmpeg(t *testing.T) {
	f := decrypt.NewFFmpeg()

	path := f.Resolve()
	if path == "" {
		t.Fatal("Resolve must return a path")
	}

	// The resolved path must be invokable.
	cmd := exec.Command(path, "-version")
	if err := cmd.Run(); err != nil {
		t.Skipf("ffmpeg is not available: %v", err)
	}
}

func TestResolveFFprobeFindsFFprobe(t *testing.T) {
	f := decrypt.NewFFmpeg()

	path := f.ResolveFFprobe()
	if path == "" {
		t.Fatal("ResolveFFprobe must return a path")
	}

	cmd := exec.Command(path, "-version")
	if err := cmd.Run(); err != nil {
		t.Skipf("ffprobe is not available: %v", err)
	}
}

func TestChannelsReportsTheChannelCount(t *testing.T) {
	ffmpeg := exec.Command("ffmpeg", "-version")
	if err := ffmpeg.Run(); err != nil {
		t.Skip("ffmpeg is not available")
	}

	// Generate a stereo test file.
	dir := t.TempDir()
	wav := filepath.Join(dir, "test.wav")
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-ac", "2", "-ar", "44100", wav)
	if err := cmd.Run(); err != nil {
		t.Skipf("cannot generate a test file: %v", err)
	}

	f := decrypt.NewFFmpeg()
	n, err := f.Channels(context.Background(), wav)
	if err != nil {
		t.Fatalf("Channels failed: %v", err)
	}
	if n != 2 {
		t.Errorf("the channel count is %d, want 2", n)
	}
}

func TestChannelsReportsMono(t *testing.T) {
	ffmpeg := exec.Command("ffmpeg", "-version")
	if err := ffmpeg.Run(); err != nil {
		t.Skip("ffmpeg is not available")
	}

	dir := t.TempDir()
	wav := filepath.Join(dir, "mono.wav")
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-ac", "1", "-ar", "44100", wav)
	if err := cmd.Run(); err != nil {
		t.Skipf("cannot generate a test file: %v", err)
	}

	f := decrypt.NewFFmpeg()
	n, err := f.Channels(context.Background(), wav)
	if err != nil {
		t.Fatalf("Channels failed: %v", err)
	}
	if n != 1 {
		t.Errorf("the channel count is %d, want 1", n)
	}
}

func TestDecryptMissingFileReturnsAnError(t *testing.T) {
	f := decrypt.NewFFmpeg()

	err := f.Decrypt(context.Background(), "does-not-exist.aaxc", filepath.Join(t.TempDir(), "out.m4b"), "00", "00", 100, nil)
	if err == nil {
		t.Error("Decrypt must fail when the input does not exist")
	}
}


