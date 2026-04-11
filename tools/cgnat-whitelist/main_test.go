package main

import (
	"os"
	"strings"
	"testing"
)

func TestReadCallsigns(t *testing.T) {
	// Create a temporary file with test callsigns
	tmpFile, err := os.CreateTemp("", "callsigns_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write test data
	testData := `# This is a comment
KF8S
KE8VSI
; Another comment

W1ABC
# Empty line above and below

N0CALL
`
	if _, err := tmpFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	_ = tmpFile.Close()

	// Read callsigns
	callsigns, err := readCallsigns(tmpFile.Name())
	if err != nil {
		t.Fatalf("readCallsigns failed: %v", err)
	}

	// Verify results
	expected := []string{"KF8S", "KE8VSI", "W1ABC", "N0CALL"}
	if len(callsigns) != len(expected) {
		t.Errorf("Expected %d callsigns, got %d", len(expected), len(callsigns))
	}

	for i, exp := range expected {
		if i >= len(callsigns) {
			t.Errorf("Missing callsign at index %d: expected %s", i, exp)
			continue
		}
		if callsigns[i] != exp {
			t.Errorf("At index %d: expected %s, got %s", i, exp, callsigns[i])
		}
	}
}

func TestReadCallsigns_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "callsigns_empty_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	callsigns, err := readCallsigns(tmpFile.Name())
	if err != nil {
		t.Fatalf("readCallsigns failed: %v", err)
	}

	if len(callsigns) != 0 {
		t.Errorf("Expected 0 callsigns from empty file, got %d", len(callsigns))
	}
}

func TestReadCallsigns_OnlyComments(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "callsigns_comments_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	testData := `# Comment line 1
; Comment line 2
# Comment line 3
`
	if _, err := tmpFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	_ = tmpFile.Close()

	callsigns, err := readCallsigns(tmpFile.Name())
	if err != nil {
		t.Fatalf("readCallsigns failed: %v", err)
	}

	if len(callsigns) != 0 {
		t.Errorf("Expected 0 callsigns from comments-only file, got %d", len(callsigns))
	}
}

func TestReadCallsigns_WithExtraWhitespace(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "callsigns_whitespace_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	testData := `  KF8S  
	KE8VSI	
W1ABC    some extra text here
  N0CALL
`
	if _, err := tmpFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	_ = tmpFile.Close()

	callsigns, err := readCallsigns(tmpFile.Name())
	if err != nil {
		t.Fatalf("readCallsigns failed: %v", err)
	}

	expected := []string{"KF8S", "KE8VSI", "W1ABC", "N0CALL"}
	if len(callsigns) != len(expected) {
		t.Errorf("Expected %d callsigns, got %d", len(expected), len(callsigns))
	}

	for i, exp := range expected {
		if i >= len(callsigns) {
			t.Errorf("Missing callsign at index %d: expected %s", i, exp)
			continue
		}
		if callsigns[i] != exp {
			t.Errorf("At index %d: expected %s, got %s", i, exp, callsigns[i])
		}
	}
}

func TestReadCallsigns_CaseConversion(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "callsigns_case_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	testData := `kf8s
Ke8vsi
W1abc
n0call
`
	if _, err := tmpFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	_ = tmpFile.Close()

	callsigns, err := readCallsigns(tmpFile.Name())
	if err != nil {
		t.Fatalf("readCallsigns failed: %v", err)
	}

	// All callsigns should be converted to uppercase
	for i, callsign := range callsigns {
		if callsign != strings.ToUpper(callsign) {
			t.Errorf("Callsign at index %d not uppercase: %s", i, callsign)
		}
	}
}

func TestReadCallsigns_NonExistentFile(t *testing.T) {
	_, err := readCallsigns("/this/file/does/not/exist.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
