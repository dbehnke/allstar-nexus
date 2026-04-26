package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Snapshot struct {
	Host        string
	Username    string
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	TotalEvents int
	EventCounts map[string]int
	Events      []*AMIEvent
}

type countPair struct {
	EventType string
	Count     int
}

func writeSnapshot(rb *RingBuffer, host, username string, startTime time.Time, outputPath string) (string, error) {
	events := rb.GetAll()
	endTime := time.Now()

	eventCounts := make(map[string]int)
	for _, e := range events {
		eventCounts[e.Type]++
	}

	if outputPath == "" {
		timestamp := startTime.Format("20060102-150405")
		outputPath = fmt.Sprintf("ami-debug-%s.log", timestamp)
	}

	outputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create snapshot file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close snapshot file: %w", cerr)
		}
	}()

	writef := func(format string, args ...any) {
		if err != nil {
			return
		}
		_, werr := fmt.Fprintf(f, format, args...)
		if werr != nil {
			err = fmt.Errorf("write snapshot: %w", werr)
		}
	}

	writef("=== AMI Debug Capture ===\n")
	writef("Host: %s\n", host)
	writef("Username: %s\n", username)
	writef("Started: %s\n", startTime.Format(time.RFC3339))
	writef("Ended: %s\n", endTime.Format(time.RFC3339))
	writef("Duration: %s\n", endTime.Sub(startTime).Round(time.Second))
	writef("Total Events: %d\n", len(events))
	writef("\n")

	writef("--- Event Summary ---\n")
	if len(eventCounts) == 0 {
		writef("(no events captured)\n")
	} else {
		var pairs []countPair
		for eventType, count := range eventCounts {
			pairs = append(pairs, countPair{EventType: eventType, Count: count})
		}
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].Count > pairs[j].Count
		})

		for _, pair := range pairs {
			writef("%s: %d\n", pair.EventType, pair.Count)
		}
	}
	writef("\n")

	writef("--- Raw Frames ---\n")
	for _, event := range events {
		writef("[%s] Event: %s\n", event.Timestamp.Format(time.RFC3339), event.Type)
		for key, value := range event.Headers {
			writef("%s: %s\n", key, value)
		}
		writef("\n")
	}

	return outputPath, err
}
