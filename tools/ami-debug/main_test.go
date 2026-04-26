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

func TestRingBuffer(t *testing.T) {
	rb := newRingBuffer(3)

	e1 := &AMIEvent{Type: "VarSet"}
	e2 := &AMIEvent{Type: "Newchannel"}
	e3 := &AMIEvent{Type: "Hangup"}
	e4 := &AMIEvent{Type: "Dial"}

	rb.Add(e1)
	rb.Add(e2)
	rb.Add(e3)

	events := rb.GetAll()
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}

	rb.Add(e4)
	events = rb.GetAll()
	if len(events) != 3 {
		t.Errorf("expected 3 events after overflow, got %d", len(events))
	}
	if events[0].Type != "Newchannel" {
		t.Errorf("expected first event to be Newchannel after overflow, got %s", events[0].Type)
	}
	if events[2].Type != "Dial" {
		t.Errorf("expected last event to be Dial, got %s", events[2].Type)
	}
}

func TestRingBufferEmpty(t *testing.T) {
	rb := newRingBuffer(10)
	events := rb.GetAll()
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestRingBufferCount(t *testing.T) {
	rb := newRingBuffer(5)
	if rb.Count() != 0 {
		t.Errorf("expected count 0, got %d", rb.Count())
	}

	rb.Add(&AMIEvent{Type: "Test"})
	if rb.Count() != 1 {
		t.Errorf("expected count 1, got %d", rb.Count())
	}
}
