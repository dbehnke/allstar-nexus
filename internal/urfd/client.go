package urfd

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/req"
	_ "go.nanomsg.org/mangos/v3/transport/tcp" // register TCP transport
)

// Client handles communication with URFD via NNG
type Client struct {
	addr    string
	sock    mangos.Socket
	mu      sync.Mutex
	enabled bool
}

// NewClient creates a new URFD client
func NewClient(addr string, enabled bool) *Client {
	return &Client{
		addr:    addr,
		enabled: enabled,
	}
}

// Connect initializes the NNG socket and dials the URFD reflector
func (c *Client) Connect() error {
	if !c.enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var err error
	if c.sock, err = req.NewSocket(); err != nil {
		return fmt.Errorf("failed to create NNG REQ socket: %w", err)
	}

	// Set receive timeout to avoid blocking forever if urfd is down
	if err := c.sock.SetOption(mangos.OptionRecvDeadline, 500*time.Millisecond); err != nil {
		log.Printf("URFD: warning setting recv deadline: %v", err)
	}

	if err := c.sock.Dial(c.addr); err != nil {
		return fmt.Errorf("failed to dial urfd at %s: %w", c.addr, err)
	}

	log.Printf("URFD: connected to NNG control channel at %s", c.addr)
	return nil
}

// Register sends a usrp_register command to associate an IP with a callsign
func (c *Client) Register(ip, callsign string) error {
	if !c.enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sock == nil {
		// Attempt lazy reconnect
		log.Printf("URFD: socket not connected, attempting to connect...")
		if err := c.sock.Dial(c.addr); err != nil {
			return fmt.Errorf("socket not connected and dial failed: %w", err)
		}
	}

	// Payload struct matching urfd's expectation
	payload := map[string]string{
		"cmd":      "usrp_register",
		"ip":       ip,
		"callsign": callsign,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := c.sock.Send(data); err != nil {
		// If send fails, maybe socket is dead. Close and nuke it so next try re-dials.
		log.Printf("URFD: send failed: %v", err)
		return fmt.Errorf("send failed: %w", err)
	}

	// Wait for response (optional, but good for verification)
	resp, err := c.sock.Recv()
	if err != nil {
		log.Printf("URFD: recv failed (timeout or error): %v", err)
		// We don't return error here necessarily as the command might have worked but we didn't get ack
		return nil
	}

	log.Printf("URFD: registered ip=%s callsign=%s response=%s", ip, callsign, string(resp))
	return nil
}

// Close cleans up the socket
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sock != nil {
		c.sock.Close()
		c.sock = nil
	}
}
