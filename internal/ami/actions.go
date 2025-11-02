package ami

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// ActionResult represents the result of an AMI action
type ActionResult struct {
	Success bool
	Message string
	Headers map[string]string
	Error   error
}

// generateActionID creates a unique action ID
func generateActionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "act_" + hex.EncodeToString(bytes)
}

// ExecuteAction sends a synchronous AMI action and waits for response
func (c *Connector) ExecuteAction(action string, params map[string]string, timeout time.Duration) (*ActionResult, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("AMI not connected")
	}

	if timeout == 0 {
		timeout = 10 * time.Second
	}

	actionID := generateActionID()

	// Create response channel
	responseChan := make(chan Message, 1)

	// Register pending action
	c.actionMu.Lock()
	c.pending[actionID] = responseChan
	c.actionMu.Unlock()

	// Clean up on exit
	defer func() {
		c.actionMu.Lock()
		delete(c.pending, actionID)
		c.actionMu.Unlock()
	}()

	// Build action command
	var cmd strings.Builder
	cmd.WriteString("Action: ")
	cmd.WriteString(action)
	cmd.WriteString("\r\n")
	cmd.WriteString("ActionID: ")
	cmd.WriteString(actionID)
	cmd.WriteString("\r\n")

	for key, value := range params {
		cmd.WriteString(key)
		cmd.WriteString(": ")
		cmd.WriteString(value)
		cmd.WriteString("\r\n")
	}
	cmd.WriteString("\r\n")

	// Send command
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("no active connection")
	}

	if _, err := conn.Write([]byte(cmd.String())); err != nil {
		return nil, fmt.Errorf("failed to send action: %w", err)
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		result := &ActionResult{
			Headers: response.Headers,
		}

		// Check response status
		if status, ok := response.Headers["Response"]; ok {
			result.Success = strings.ToLower(status) == "success"
			result.Message = response.Headers["Message"]
		} else {
			result.Success = false
			result.Message = "No response status"
		}

		return result, nil

	case <-time.After(timeout):
		return nil, fmt.Errorf("action timeout after %s", timeout)
	}
}

// RptCmd executes an Asterisk rpt command (e.g., "rpt fun <node> *3<remotenode>")
func (c *Connector) RptCmd(command string) (*ActionResult, error) {
	return c.ExecuteAction("Command", map[string]string{
		"Command": command,
	}, 15*time.Second)
}

// LinkNode links a remote node to a local node
func (c *Connector) LinkNode(localNode int, remoteNode int, permanent bool) (*ActionResult, error) {
	mode := "3" // Transceive
	if permanent {
		mode = "2" // Permanent
	}

	command := fmt.Sprintf("rpt fun %d *%s%d", localNode, mode, remoteNode)
	return c.RptCmd(command)
}

// UnlinkNode disconnects a remote node
func (c *Connector) UnlinkNode(localNode int, remoteNode int) (*ActionResult, error) {
	command := fmt.Sprintf("rpt fun %d *1%d", localNode, remoteNode)
	return c.RptCmd(command)
}

// GetNodeStatus executes "rpt lstats <node>" to get node statistics
func (c *Connector) GetNodeStatus(node int) (*ActionResult, error) {
	command := fmt.Sprintf("rpt lstats %d", node)
	return c.RptCmd(command)
}

// Originate initiates an outbound call
func (c *Connector) Originate(channel, context, exten, priority string, timeout int) (*ActionResult, error) {
	params := map[string]string{
		"Channel":  channel,
		"Context":  context,
		"Exten":    exten,
		"Priority": priority,
	}

	if timeout > 0 {
		params["Timeout"] = fmt.Sprintf("%d", timeout*1000) // Convert to milliseconds
	}

	return c.ExecuteAction("Originate", params, 30*time.Second)
}

// Monitor starts monitoring a channel
func (c *Connector) Monitor(channel, file string, format string, mix bool) (*ActionResult, error) {
	params := map[string]string{
		"Channel": channel,
		"File":    file,
	}

	if format != "" {
		params["Format"] = format
	}

	if mix {
		params["Mix"] = "true"
	}

	return c.ExecuteAction("Monitor", params, 10*time.Second)
}

// Reload reloads Asterisk configuration
func (c *Connector) Reload(module string) (*ActionResult, error) {
	params := map[string]string{}
	if module != "" {
		params["Module"] = module
	}

	return c.ExecuteAction("Reload", params, 30*time.Second)
}

// CoreShowChannels retrieves active channels
func (c *Connector) CoreShowChannels() (*ActionResult, error) {
	return c.ExecuteAction("CoreShowChannels", map[string]string{}, 15*time.Second)
}

// Ping sends a ping to verify connection
func (c *Connector) Ping() (*ActionResult, error) {
	return c.ExecuteAction("Ping", map[string]string{}, 5*time.Second)
}
