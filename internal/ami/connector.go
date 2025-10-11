package ami

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MessageType enumerates classification of AMI frames.
type MessageType string

const (
	MessageTypeUnknown  MessageType = "UNKNOWN"
	MessageTypeEvent    MessageType = "EVENT"
	MessageTypeResponse MessageType = "RESPONSE"
)

// Message represents a generic AMI frame (Event or Response) with headers.
type Message struct {
	Type    MessageType
	Headers map[string]string
	Raw     []string
}

// Snapshot minimal exported state placeholder (will expand later).
type Snapshot struct {
	Timestamp time.Time
	Connected bool
}

// ConnectionStatus represents the AMI connection state
type ConnectionStatus struct {
	Connected bool
	Timestamp time.Time
	Error     error
}

// Connector manages AMI TCP connection, login, reconnect, and frame parsing.
type Connector struct {
	host     string
	port     int
	user     string
	pass     string
	events   string
	retryMin time.Duration
	retryMax time.Duration

	mu        sync.RWMutex
	running   bool
	conn      net.Conn
	connected bool // track connection state

	rawOut    chan Message          // public channel for downstream consumption
	statusOut chan ConnectionStatus // connection status changes

	// debug counter: number of Event/VarSet frames logged so far
	debugEventLogged int

	actionMu sync.Mutex
	pending  map[string]chan Message // ActionID -> single-response channel
}

// NewConnector builds a connector (not started yet).
func NewConnector(host string, port int, user, pass, events string, retryMin, retryMax time.Duration) *Connector {
	return &Connector{
		host:      host,
		port:      port,
		user:      user,
		pass:      pass,
		events:    events,
		retryMin:  retryMin,
		retryMax:  retryMax,
		rawOut:    make(chan Message),
		statusOut: make(chan ConnectionStatus, 4),
		pending:   make(map[string]chan Message),
	}
}

// Raw returns the channel of parsed AMI messages.
func (c *Connector) Raw() <-chan Message { return c.rawOut }

// ConnectionStatus returns the channel of connection status changes.
func (c *Connector) ConnectionStatusChan() <-chan ConnectionStatus { return c.statusOut }

// IsConnected returns the current connection state.
func (c *Connector) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Start launches connection management loop.
func (c *Connector) Start(ctx context.Context) error {
	if !c.setRunning() {
		return fmt.Errorf("connector already running")
	}
	// Log that Start was invoked
	log.Printf("[AMI] Start invoked")
	go c.loop(ctx)
	return nil
}

// setRunning atomic check/set.
func (c *Connector) setRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return false
	}
	c.running = true
	return true
}

func (c *Connector) loop(ctx context.Context) {
	backoff := c.retryMin
	if backoff <= 0 {
		backoff = 5 * time.Second
	}
	attemptCount := 0

	for {
		if ctx.Err() != nil {
			log.Printf("[AMI] connection loop stopped (context cancelled)")
			return
		}

		attemptCount++
		// Log connection attempt with backoff info
		if attemptCount == 1 {
			log.Printf("[AMI] attempting initial connection to %s:%d", c.host, c.port)
		} else {
			log.Printf("[AMI] reconnection attempt #%d to %s:%d (backoff: %s)", attemptCount, c.host, c.port, backoff)
		}

		if err := c.connectAndServe(ctx); err != nil {
			log.Printf("[AMI] connection failed (attempt #%d): %v", attemptCount, err)

			// Broadcast disconnection status
			c.broadcastStatus(false, err)

			// Clear any pending actions
			c.clearPendingActions()
		} else {
			// Successful disconnection (graceful shutdown) - reset attempt count
			log.Printf("[AMI] connection closed gracefully")
			attemptCount = 0
		}

		// Wait with exponential backoff before retry
		select {
		case <-ctx.Done():
			log.Printf("[AMI] connection loop stopped (context cancelled during backoff)")
			return
		case <-time.After(backoff):
			// Double the backoff for next attempt
			nextBackoff := backoff * 2
			if nextBackoff > c.retryMax && c.retryMax > 0 {
				nextBackoff = c.retryMax
			}

			// Reset backoff on successful connection (when attempt count resets)
			if attemptCount == 0 {
				backoff = c.retryMin
				if backoff <= 0 {
					backoff = 5 * time.Second
				}
			} else {
				backoff = nextBackoff
			}

			log.Printf("[AMI] retrying connection in %s...", backoff)
		}
	}
}

func (c *Connector) connectAndServe(ctx context.Context) error {
	addr := net.JoinHostPort(c.host, strconv.Itoa(c.port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// Log successful TCP connection so it's visible in server logs
	log.Printf("[AMI] connected to %s", addr)
	if err := c.sendLogin(); err != nil {
		conn.Close()
		return err
	}

	// Login payload sent successfully
	log.Printf("[AMI] login payload sent to %s", addr)

	// Broadcast successful connection status
	c.broadcastStatus(true, nil)
	reader := bufio.NewReader(conn)
	var frame []string
	flush := func() error {
		if len(frame) == 0 {
			return nil
		}
		headers := make(map[string]string, len(frame))
		for _, ln := range frame {
			if idx := strings.Index(ln, ":"); idx > 0 {
				k := strings.TrimSpace(ln[:idx])
				v := strings.TrimSpace(ln[idx+1:])
				headers[k] = v
			}
		}
		mtype := MessageTypeUnknown
		if _, ok := headers["Event"]; ok {
			mtype = MessageTypeEvent
		} else if _, ok := headers["Response"]; ok {
			mtype = MessageTypeResponse
		} else if _, ok := headers["ActionID"]; ok {
			mtype = MessageTypeResponse
		}

		// Diagnostic: if this is an Event or VarSet frame, log the raw frame (limited to 50)
		if (mtype == MessageTypeEvent || headers["Event"] == "VarSet") && c.debugEventLogged < 50 {
			log.Printf("[AMI DEBUG] received frame headers: %v", headers)
			for i, ln := range frame {
				log.Printf("[AMI DEBUG] raw[%d]: %s", i, ln)
			}
			c.debugEventLogged++
		}

		// Diagnostic: log frames that look like RPT events (headers contain RPT_ keys or raw contains RPT_ text)
		if c.debugEventLogged < 200 { // overall throttle
			foundRPT := false
			for k := range headers {
				if strings.HasPrefix(k, "RPT_") {
					foundRPT = true
					break
				}
			}
			if !foundRPT {
				for _, ln := range frame {
					if strings.Contains(ln, "RPT_") || strings.Contains(ln, "RPT ") {
						foundRPT = true
						break
					}
				}
			}
			if foundRPT {
				log.Printf("[AMI RPT] detected RPT frame headers: %v", headers)
				for i, ln := range frame {
					log.Printf("[AMI RPT] raw[%d]: %s", i, ln)
				}
				c.debugEventLogged += 5
			}
		}
		msg := Message{Type: mtype, Headers: headers, Raw: append([]string(nil), frame...)}
		frame = frame[:0]
		// Action correlation
		if id, ok := headers["ActionID"]; ok {
			c.actionMu.Lock()
			ch, found := c.pending[id]
			if found {
				delete(c.pending, id)
			}
			c.actionMu.Unlock()
			if found {
				select {
				case ch <- msg:
				default:
				}
			}
		}
		select {
		case c.rawOut <- msg:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		frame = append(frame, line)
	}
}

func (c *Connector) sendLogin() error {
	actionID := randID()
	payload := fmt.Sprintf("Action: Login\r\nActionID: %s\r\nUsername: %s\r\nSecret: %s\r\nEvents: %s\r\n\r\n", actionID, c.user, c.pass, c.events)
	_, err := c.conn.Write([]byte(payload))
	return err
}

// SendCommand issues a generic AMI CLI command (Action: Command) and waits for response correlated by ActionID.
func (c *Connector) SendCommand(ctx context.Context, command string) (Message, error) {
	id := randID()
	ch := make(chan Message, 1)
	c.actionMu.Lock()
	c.pending[id] = ch
	c.actionMu.Unlock()
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return Message{}, fmt.Errorf("not connected")
	}
	payload := fmt.Sprintf("Action: Command\r\nActionID: %s\r\nCommand: %s\r\n\r\n", id, command)
	if _, err := conn.Write([]byte(payload)); err != nil {
		return Message{}, err
	}
	select {
	case msg := <-ch:
		return msg, nil
	case <-ctx.Done():
		c.actionMu.Lock()
		delete(c.pending, id)
		c.actionMu.Unlock()
		return Message{}, ctx.Err()
	}
}

// RptStatus issues an RptStatus command with specified subcommand (XStat, SawStat, etc.)
func (c *Connector) RptStatus(ctx context.Context, node int, command string) (Message, error) {
	id := randID()
	ch := make(chan Message, 1)
	c.actionMu.Lock()
	c.pending[id] = ch
	c.actionMu.Unlock()
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return Message{}, fmt.Errorf("not connected")
	}
	payload := fmt.Sprintf("Action: RptStatus\r\nActionID: %s\r\nCommand: %s\r\nNode: %d\r\n\r\n", id, command, node)
	if _, err := conn.Write([]byte(payload)); err != nil {
		return Message{}, err
	}
	select {
	case msg := <-ch:
		return msg, nil
	case <-ctx.Done():
		c.actionMu.Lock()
		delete(c.pending, id)
		c.actionMu.Unlock()
		return Message{}, ctx.Err()
	}
}

// GetXStat retrieves and parses XStat for a node
func (c *Connector) GetXStat(ctx context.Context, node int) (*XStatResult, error) {
	msg, err := c.RptStatus(ctx, node, "XStat")
	if err != nil {
		return nil, err
	}

	// Extract response text from message
	responseText := extractCommandOutput(msg)
	return ParseXStat(node, responseText)
}

// GetSawStat retrieves and parses SawStat for a node
func (c *Connector) GetSawStat(ctx context.Context, node int) (*SawStatResult, error) {
	msg, err := c.RptStatus(ctx, node, "SawStat")
	if err != nil {
		return nil, err
	}

	// Extract response text from message
	responseText := extractCommandOutput(msg)
	return ParseSawStat(node, responseText)
}

// GetCombinedStatus retrieves both XStat and SawStat and combines them
func (c *Connector) GetCombinedStatus(ctx context.Context, node int) (*CombinedNodeStatus, error) {
	// Get XStat
	xstat, err := c.GetXStat(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("xstat failed: %w", err)
	}

	// Get SawStat
	sawstat, err := c.GetSawStat(ctx, node)
	if err != nil {
		// Don't fail if SawStat fails - just use nil
		sawstat = nil
	}

	// Combine results
	return CombineXStatSawStat(xstat, sawstat), nil
}

// extractCommandOutput extracts the command output from an AMI response
func extractCommandOutput(msg Message) string {
	// The response is in msg.Raw
	var output []string
	inOutput := false

	for _, line := range msg.Raw {
		// Skip until we see the response start
		if strings.Contains(line, "Response:") || strings.Contains(line, "Message:") {
			inOutput = true
			continue
		}

		// Stop at end marker
		if strings.Contains(line, "--END COMMAND--") {
			break
		}

		if inOutput {
			output = append(output, line)
		}
	}

	return strings.Join(output, "\n")
}

// broadcastStatus sends connection status to statusOut channel
func (c *Connector) broadcastStatus(connected bool, err error) {
	c.mu.Lock()
	c.connected = connected
	c.mu.Unlock()

	select {
	case c.statusOut <- ConnectionStatus{Connected: connected, Timestamp: time.Now(), Error: err}:
	default:
	}
}

// clearPendingActions clears all pending action responses
func (c *Connector) clearPendingActions() {
	c.actionMu.Lock()
	defer c.actionMu.Unlock()
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
}

func randID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexdigits[v>>4]
		out[i*2+1] = hexdigits[v&0x0f]
	}
	return string(out)
}
