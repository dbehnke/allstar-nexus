package admin

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dbehnke/allstar-nexus/internal/ami"
)

// CommandExecutor handles safe execution of AMI commands
type CommandExecutor struct {
	AMI          *ami.Connector
	AllowedNodes []int
	SafetyMode   bool // If true, only allow whitelisted commands
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(amiConn *ami.Connector, allowedNodes []int) *CommandExecutor {
	return &CommandExecutor{
		AMI:          amiConn,
		AllowedNodes: allowedNodes,
		SafetyMode:   true,
	}
}

// CommandResult represents the result of command execution
type CommandResult struct {
	Success   bool      `json:"success"`
	Output    string    `json:"output"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Duration  string    `json:"duration"`
}

// ExecuteCommand executes a raw AMI command with safety checks
func (ce *CommandExecutor) ExecuteCommand(command string) (*CommandResult, error) {
	startTime := time.Now()

	result := &CommandResult{
		Timestamp: startTime,
	}

	// Safety checks
	if err := ce.validateCommand(command); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	// Execute via AMI
	amiResult, err := ce.AMI.RptCmd(command)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	result.Success = amiResult.Success
	result.Output = amiResult.Message
	if !amiResult.Success && amiResult.Message != "" {
		result.Error = amiResult.Message
	}
	result.Duration = time.Since(startTime).String()

	return result, nil
}

// LinkNode links two nodes together
func (ce *CommandExecutor) LinkNode(localNode, remoteNode int, permanent bool) (*CommandResult, error) {
	startTime := time.Now()

	result := &CommandResult{
		Timestamp: startTime,
	}

	// Validate local node is allowed
	if !ce.isNodeAllowed(localNode) {
		err := fmt.Errorf("node %d not in allowed list", localNode)
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	amiResult, err := ce.AMI.LinkNode(localNode, remoteNode, permanent)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	result.Success = amiResult.Success
	result.Output = fmt.Sprintf("Linked node %d to %d (permanent: %v)", localNode, remoteNode, permanent)
	if !amiResult.Success && amiResult.Message != "" {
		result.Error = amiResult.Message
	}
	result.Duration = time.Since(startTime).String()

	return result, nil
}

// UnlinkNode disconnects two nodes
func (ce *CommandExecutor) UnlinkNode(localNode, remoteNode int) (*CommandResult, error) {
	startTime := time.Now()

	result := &CommandResult{
		Timestamp: startTime,
	}

	// Validate local node is allowed
	if !ce.isNodeAllowed(localNode) {
		err := fmt.Errorf("node %d not in allowed list", localNode)
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	amiResult, err := ce.AMI.UnlinkNode(localNode, remoteNode)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	result.Success = amiResult.Success
	result.Output = fmt.Sprintf("Unlinked node %d from %d", localNode, remoteNode)
	if !amiResult.Success && amiResult.Message != "" {
		result.Error = amiResult.Message
	}
	result.Duration = time.Since(startTime).String()

	return result, nil
}

// GetNodeStatus retrieves status for a node
func (ce *CommandExecutor) GetNodeStatus(node int) (*CommandResult, error) {
	startTime := time.Now()

	result := &CommandResult{
		Timestamp: startTime,
	}

	amiResult, err := ce.AMI.GetNodeStatus(node)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	result.Success = amiResult.Success
	result.Output = amiResult.Message
	if !amiResult.Success && amiResult.Message != "" {
		result.Error = amiResult.Message
	}
	result.Duration = time.Since(startTime).String()

	return result, nil
}

// Reload reloads Asterisk configuration
func (ce *CommandExecutor) Reload(module string) (*CommandResult, error) {
	startTime := time.Now()

	result := &CommandResult{
		Timestamp: startTime,
	}

	amiResult, err := ce.AMI.Reload(module)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	result.Success = amiResult.Success
	if module != "" {
		result.Output = fmt.Sprintf("Reloaded module: %s", module)
	} else {
		result.Output = "Reloaded Asterisk configuration"
	}
	if !amiResult.Success && amiResult.Message != "" {
		result.Error = amiResult.Message
	}
	result.Duration = time.Since(startTime).String()

	return result, nil
}

// validateCommand checks if a command is safe to execute
func (ce *CommandExecutor) validateCommand(command string) error {
	if command == "" {
		return fmt.Errorf("empty command")
	}

	// Trim whitespace
	command = strings.TrimSpace(command)

	// In safety mode, only allow whitelisted command patterns
	if ce.SafetyMode {
		allowed := []string{
			`^rpt\s+fun\s+\d+\s+\*[123]\d+$`,        // Link/unlink commands
			`^rpt\s+lstats\s+\d+$`,                   // Node stats
			`^rpt\s+nodes\s+\d+$`,                    // Show nodes
			`^rpt\s+xnode\s+\d+$`,                    // Extended node info
			`^rpt\s+sawstat\s+\d+$`,                  // SAW stats
			`^rpt\s+xstat\s+\d+$`,                    // Extended stats
			`^core\s+show\s+channels`,                // Show channels
			`^module\s+reload\s+\w+`,                 // Reload modules
		}

		matched := false
		for _, pattern := range allowed {
			if matched, _ = regexp.MatchString(pattern, command); matched {
				break
			}
		}

		if !matched {
			return fmt.Errorf("command not in whitelist (safety mode enabled)")
		}
	}

	// Block dangerous commands regardless of safety mode
	dangerous := []string{
		"core restart",
		"core shutdown",
		"stop now",
		"core stop",
		"database del",
		"database deltree",
	}

	commandLower := strings.ToLower(command)
	for _, danger := range dangerous {
		if strings.Contains(commandLower, danger) {
			return fmt.Errorf("dangerous command blocked: %s", danger)
		}
	}

	return nil
}

// isNodeAllowed checks if a node is in the allowed list
func (ce *CommandExecutor) isNodeAllowed(node int) bool {
	if len(ce.AllowedNodes) == 0 {
		return true // No restrictions if list is empty
	}

	for _, allowed := range ce.AllowedNodes {
		if allowed == node {
			return true
		}
	}
	return false
}

// GetPredefinedCommands returns a list of common safe commands
func (ce *CommandExecutor) GetPredefinedCommands() []PredefinedCommand {
	commands := []PredefinedCommand{
		{
			ID:          "link_transceive",
			Label:       "Link Node (Transceive)",
			Description: "Connect to another node with full duplex",
			Template:    "rpt fun {local} *3{remote}",
			Category:    "linking",
			RequiresConfirm: false,
		},
		{
			ID:          "link_monitor",
			Label:       "Link Node (Monitor Only)",
			Description: "Connect to another node in receive-only mode",
			Template:    "rpt fun {local} *2{remote}",
			Category:    "linking",
			RequiresConfirm: false,
		},
		{
			ID:          "unlink",
			Label:       "Unlink Node",
			Description: "Disconnect from a connected node",
			Template:    "rpt fun {local} *1{remote}",
			Category:    "linking",
			RequiresConfirm: false,
		},
		{
			ID:          "unlink_all",
			Label:       "Unlink All Nodes",
			Description: "Disconnect all connected nodes",
			Template:    "rpt fun {local} *76",
			Category:    "linking",
			RequiresConfirm: true,
		},
		{
			ID:          "node_status",
			Label:       "Node Status",
			Description: "Show detailed node statistics",
			Template:    "rpt lstats {local}",
			Category:    "status",
			RequiresConfirm: false,
		},
		{
			ID:          "show_nodes",
			Label:       "Show Connected Nodes",
			Description: "List all currently connected nodes",
			Template:    "rpt nodes {local}",
			Category:    "status",
			RequiresConfirm: false,
		},
		{
			ID:          "reload_rpt",
			Label:       "Reload app_rpt",
			Description: "Reload the app_rpt module configuration",
			Template:    "module reload app_rpt.so",
			Category:    "admin",
			RequiresConfirm: true,
		},
	}

	return commands
}

// PredefinedCommand represents a pre-defined safe command template
type PredefinedCommand struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	Description     string `json:"description"`
	Template        string `json:"template"`
	Category        string `json:"category"`
	RequiresConfirm bool   `json:"requires_confirm"`
}
