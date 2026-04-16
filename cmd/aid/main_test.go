package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunReturnsRootHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exit := run([]string{"--help"}, &stdout, &stderr); exit != 0 {
		t.Fatalf("expected exit 0, got %d: stderr=%q", exit, stderr.String())
	}
	if !strings.Contains(stdout.String(), "aid - local memory for coding agents and repos") {
		t.Fatalf("unexpected help output: %q", stdout.String())
	}
}

func TestRunUsageErrorExitsWithUsageCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exit := run([]string{"--brief", "--verbose"}, &stdout, &stderr); exit != 2 {
		t.Fatalf("expected usage exit 2, got %d (stderr=%q)", exit, stderr.String())
	}
}
