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
	defer f.Close()

	fmt.Fprintf(f, "=== AMI Debug Capture ===\n")
	fmt.Fprintf(f, "Host: %s\n", host)
	fmt.Fprintf(f, "Username: %s\n", username)
	fmt.Fprintf(f, "Started: %s\n", startTime.Format(time.RFC3339))
	fmt.Fprintf(f, "Ended: %s\n", endTime.Format(time.RFC3339))
	fmt.Fprintf(f, "Duration: %s\n", endTime.Sub(startTime).Round(time.Second))
	fmt.Fprintf(f, "Total Events: %d\n", len(events))
	fmt.Fprintf(f, "\n")

	fmt.Fprintf(f, "--- Event Summary ---\n")
	if len(eventCounts) == 0 {
		fmt.Fprintf(f, "(no events captured)\n")
	} else {
		var pairs []countPair
		for eventType, count := range eventCounts {
			pairs = append(pairs, countPair{EventType: eventType, Count: count})
		}
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].Count > pairs[j].Count
		})

		for _, pair := range pairs {
			fmt.Fprintf(f, "%s: %d\n", pair.EventType, pair.Count)
		}
	}
	fmt.Fprintf(f, "\n")

	fmt.Fprintf(f, "--- Raw Frames ---\n")
	for _, event := range events {
		fmt.Fprintf(f, "[%s] Event: %s\n", event.Timestamp.Format(time.RFC3339), event.Type)
		for key, value := range event.Headers {
			fmt.Fprintf(f, "%s: %s\n", key, value)
		}
		fmt.Fprintf(f, "\n")
	}

	return outputPath, nil
}
