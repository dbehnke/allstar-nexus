package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
)

type AMIEvent struct {
	Timestamp time.Time
	Type      string
	ActionID  string
	Headers   map[string]string
	Raw       string
}

func parseFrame(reader *bufio.Reader) (*AMIEvent, error) {
	headers := make(map[string]string)
	var rawLines []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(rawLines) == 0 {
				return nil, err
			}
			if err != io.EOF {
				return nil, err
			}
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		rawLines = append(rawLines, line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers[key] = value
		}
	}

	if len(rawLines) == 0 {
		return nil, fmt.Errorf("empty frame")
	}

	event := &AMIEvent{
		Timestamp: time.Now(),
		Headers:   headers,
		Raw:       strings.Join(rawLines, "\n"),
	}

	if val, ok := headers["Event"]; ok {
		event.Type = val
	} else if val, ok := headers["Response"]; ok {
		event.Type = "Response:" + val
	}

	if val, ok := headers["ActionID"]; ok {
		event.ActionID = val
	}

	return event, nil
}
