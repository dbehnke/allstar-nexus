package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseFrame(t *testing.T) {
	input := "Event: VarSet\r\nChannel: IAX2/123\r\nVariable: RPT_ALINKS\r\nValue: 2\r\n\r\n"
	reader := bufio.NewReader(strings.NewReader(input))

	frame, err := parseFrame(reader)
	if err != nil {
		t.Fatalf("parseFrame failed: %v", err)
	}

	if frame.Type != "VarSet" {
		t.Errorf("expected Type=VarSet, got %q", frame.Type)
	}
	if frame.Headers["Channel"] != "IAX2/123" {
		t.Errorf("expected Channel=IAX2/123, got %q", frame.Headers["Channel"])
	}
	if frame.Headers["Variable"] != "RPT_ALINKS" {
		t.Errorf("expected Variable=RPT_ALINKS, got %q", frame.Headers["Variable"])
	}
	if frame.Headers["Value"] != "2" {
		t.Errorf("expected Value=2, got %q", frame.Headers["Value"])
	}
}

func TestParseFrameResponse(t *testing.T) {
	input := "Response: Success\r\nActionID: login-123\r\nMessage: Authentication accepted\r\n\r\n"
	reader := bufio.NewReader(strings.NewReader(input))

	frame, err := parseFrame(reader)
	if err != nil {
		t.Fatalf("parseFrame failed: %v", err)
	}

	if frame.Type != "Response:Success" {
		t.Errorf("expected Type=Response:Success, got %q", frame.Type)
	}
	if frame.ActionID != "login-123" {
		t.Errorf("expected ActionID=login-123, got %q", frame.ActionID)
	}
}

func TestParseFrameEmpty(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	_, err := parseFrame(reader)
	if err == nil {
		t.Error("expected error for empty input")
	}
}
