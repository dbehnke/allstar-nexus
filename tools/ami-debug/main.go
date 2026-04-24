package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("ami-debug %s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	var (
		host       = flag.String("host", "127.0.0.1:5038", "AMI host:port")
		username   = flag.String("username", "admin", "AMI username")
		duration   = flag.Duration("duration", 0, "Capture duration (0 = indefinite)")
		filter     = flag.String("filter", "", "Regex filter for event matching")
		output     = flag.String("output", "", "Snapshot output file path")
		bufferSize = flag.Int("buffer-size", 1000, "Ring buffer size for events")
		noColor    = flag.Bool("no-color", false, "Disable colored stdout output")
	)
	flag.Parse()

	password := os.Getenv("AMI_PASSWORD")
	if password == "" {
		fmt.Print("AMI Password: ")
		fmt.Scanln(&password)
		if password == "" {
			fmt.Fprintln(os.Stderr, "Error: AMI_PASSWORD environment variable not set")
			os.Exit(1)
		}
	}

	var filterRe *regexp.Regexp
	if *filter != "" {
		var err error
		filterRe, err = regexp.Compile(*filter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid filter regex: %v\n", err)
			os.Exit(1)
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Printf("Connecting to AMI at %s...\n", *host)

	var conn net.Conn
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		conn, err = net.Dial("tcp", *host)
		if err == nil {
			break
		}
		fmt.Printf("Connection attempt %d failed: %v\n", attempt, err)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect after 3 attempts: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("Connected. Logging in...\n")

	loginActionID := fmt.Sprintf("login-%d", time.Now().Unix())
	loginPayload := fmt.Sprintf("Action: Login\r\nActionID: %s\r\nUsername: %s\r\nSecret: %s\r\nEvents: on\r\n\r\n",
		loginActionID, *username, password)
	_, err = conn.Write([]byte(loginPayload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to send login: %v\n", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(conn)
	for {
		frame, err := parseFrame(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to read login response: %v\n", err)
			os.Exit(1)
		}
		if frame.ActionID == loginActionID && frame.Type == "Response:Success" {
			fmt.Printf("Login successful. Capturing events...\n")
			break
		}
		if frame.ActionID == loginActionID && frame.Type == "Response:Error" {
			fmt.Fprintf(os.Stderr, "Error: login failed: %s\n", frame.Headers["Message"])
			os.Exit(1)
		}
	}

	rb := newRingBuffer(*bufferSize)
	startTime := time.Now()

	done := make(chan struct{})
	if *duration > 0 {
		go func() {
			time.Sleep(*duration)
			close(done)
		}()
	}

	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}

			frame, err := parseFrame(reader)
			if err != nil {
				if err.Error() == "EOF" {
					fmt.Printf("\nConnection closed by server.\n")
					close(done)
					return
				}
				fmt.Fprintf(os.Stderr, "\nError reading frame: %v\n", err)
				continue
			}

			if filterRe != nil && !filterRe.MatchString(frame.Raw) {
				continue
			}

			rb.Add(frame)

			if !*noColor && isTerminal() {
				fmt.Printf("\033[36m[%s]\033[0m \033[33m%s\033[0m",
					frame.Timestamp.Format("15:04:05"), frame.Type)
			} else {
				fmt.Printf("[%s] %s", frame.Timestamp.Format("15:04:05"), frame.Type)
			}

			for key, value := range frame.Headers {
				if key != "Event" && key != "Response" {
					fmt.Printf(" | %s=%s", key, value)
				}
			}
			fmt.Println()
		}
	}()

	select {
	case <-sigChan:
		fmt.Printf("\nReceived interrupt. Saving snapshot...\n")
	case <-done:
		fmt.Printf("\nDuration reached. Saving snapshot...\n")
	}

	outputPath, err := writeSnapshot(rb, *host, *username, startTime, *output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write snapshot: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Snapshot saved to: %s\n", outputPath)
	fmt.Printf("Total events captured: %d\n", rb.Count())
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
