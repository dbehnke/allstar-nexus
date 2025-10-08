package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/config"
	"github.com/dbehnke/allstar-nexus/internal/ami"
)

// LogEntry represents a single AMI event log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Event     string                 `json:"event,omitempty"`
	Headers   map[string]string      `json:"headers"`
	Raw       []string               `json:"raw,omitempty"`
}

func main() {
	// Command-line flags
	configPath := flag.String("config", "", "Path to config.yaml (default: search standard locations)")
	outputPath := flag.String("output", "ami-events.jsonl", "Output file path (JSONL format)")
	includeRaw := flag.Bool("raw", false, "Include raw message lines in output")
	eventsOnly := flag.Bool("events-only", false, "Log only events, skip responses")
	duration := flag.Duration("duration", 0, "Stop after this duration (0 = run until interrupted)")
	verbose := flag.Bool("verbose", false, "Print events to stdout in addition to file")
	flag.Parse()

	// Load configuration from config.yaml
	cfg := config.Load(*configPath)

	if !cfg.AMIEnabled {
		log.Fatal("AMI is disabled in configuration. Please enable it to capture events.")
	}

	log.Printf("AMI Events Logger")
	log.Printf("Connecting to AMI: %s:%d (user: %s)", cfg.AMIHost, cfg.AMIPort, cfg.AMIUser)
	log.Printf("Output file: %s", *outputPath)
	log.Printf("Events only: %v", *eventsOnly)
	log.Printf("Include raw: %v", *includeRaw)
	if *duration > 0 {
		log.Printf("Duration: %v", *duration)
	} else {
		log.Printf("Duration: unlimited (press Ctrl+C to stop)")
	}

	// Open output file
	outFile, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Create JSON encoder
	encoder := json.NewEncoder(outFile)

	// Create context with optional timeout
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *duration > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), *duration)
		defer cancel()
	}

	// Handle interrupt signal for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Printf("\nReceived interrupt signal, stopping...")
		cancel()
	}()

	// Create and start AMI connector
	conn := ami.NewConnector(
		cfg.AMIHost,
		cfg.AMIPort,
		cfg.AMIUser,
		cfg.AMIPassword,
		cfg.AMIEvents,
		cfg.AMIRetryInterval,
		cfg.AMIRetryMax,
	)

	if err := conn.Start(ctx); err != nil {
		log.Fatalf("Failed to start AMI connector: %v", err)
	}

	log.Printf("Connected to AMI, capturing events...")

	// Statistics
	eventCount := 0
	responseCount := 0
	unknownCount := 0
	startTime := time.Now()

	// Process messages
	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown
			elapsed := time.Since(startTime)
			log.Printf("\nShutdown complete")
			log.Printf("Statistics:")
			log.Printf("  Duration:  %v", elapsed.Round(time.Second))
			log.Printf("  Events:    %d", eventCount)
			log.Printf("  Responses: %d", responseCount)
			log.Printf("  Unknown:   %d", unknownCount)
			log.Printf("  Total:     %d", eventCount+responseCount+unknownCount)
			log.Printf("Output saved to: %s", *outputPath)
			return

		case msg := <-conn.Raw():
			// Skip responses if events-only flag is set
			if *eventsOnly && msg.Type != ami.MessageTypeEvent {
				responseCount++
				continue
			}

			// Create log entry
			entry := LogEntry{
				Timestamp: time.Now(),
				Type:      string(msg.Type),
				Headers:   msg.Headers,
			}

			// Add event name if it's an event
			if msg.Type == ami.MessageTypeEvent {
				if eventName, ok := msg.Headers["Event"]; ok {
					entry.Event = eventName
				}
				eventCount++
			} else if msg.Type == ami.MessageTypeResponse {
				responseCount++
			} else {
				unknownCount++
			}

			// Include raw lines if requested
			if *includeRaw {
				entry.Raw = msg.Raw
			}

			// Write to file
			if err := encoder.Encode(entry); err != nil {
				log.Printf("Error encoding entry: %v", err)
				continue
			}

			// Print to stdout if verbose
			if *verbose {
				printEntry(entry)
			}
		}
	}
}

// printEntry prints a log entry in a human-readable format
func printEntry(entry LogEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05.000")
	
	if entry.Type == string(ami.MessageTypeEvent) {
		fmt.Printf("[%s] EVENT: %s\n", timestamp, entry.Event)
		
		// Print interesting headers (skip common ones)
		skipHeaders := map[string]bool{
			"Event":      true,
			"Privilege":  true,
			"ActionID":   true,
		}
		
		for k, v := range entry.Headers {
			if !skipHeaders[k] && v != "" {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}
		fmt.Println()
	} else if entry.Type == string(ami.MessageTypeResponse) {
		response := entry.Headers["Response"]
		message := entry.Headers["Message"]
		if message != "" {
			fmt.Printf("[%s] RESPONSE: %s - %s\n", timestamp, response, message)
		} else {
			fmt.Printf("[%s] RESPONSE: %s\n", timestamp, response)
		}
	} else {
		fmt.Printf("[%s] UNKNOWN MESSAGE\n", timestamp)
	}
}
