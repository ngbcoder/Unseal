package main

import (
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	if err := run([]string{"version"}); err != nil {
		t.Fatalf("version failed: %v", err)
	}
}

func TestHelpCommand(t *testing.T) {
	if err := run([]string{"help"}); err != nil {
		t.Fatalf("help failed: %v", err)
	}
}

func TestUnknownCommandReturnsAnError(t *testing.T) {
	err := run([]string{"does-not-exist"})
	if err == nil {
		t.Fatal("an unknown command must return an error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("the error is %q, want it to mention 'unknown command'", err.Error())
	}
}

func TestNoArgsPrintsUsage(t *testing.T) {
	if err := run([]string{}); err != nil {
		t.Fatalf("no args must not return an error: %v", err)
	}
}

func TestNotYetImplementedCommandsReturnAnError(t *testing.T) {
	for _, cmd := range []string{"register", "library", "download", "export", "config", "profiles"} {
		err := run([]string{cmd})
		if err == nil {
			t.Errorf("%s must return an error until it is implemented", cmd)
		}
	}
}
