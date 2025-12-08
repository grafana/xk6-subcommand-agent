// Package initagents provides k6 subcommand for initializing AI agent configurations.
package initagents

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.k6.io/k6/cmd/state"
	"go.k6.io/k6/subcommand"

	"github.com/grafana/xk6-agent/agents"
	"github.com/grafana/xk6-agent/agents/claude"
	"github.com/grafana/xk6-agent/agents/opencode"
	"github.com/grafana/xk6-agent/agents/vscode"
)

var supportedPlatforms = []string{"claude", "vscode", "opencode"}

type initializerFactory func() agents.Initializer

var defaultInitializerFactories = map[string]initializerFactory{
	"claude": func() agents.Initializer {
		return claude.NewCode()
	},
	"vscode": func() agents.Initializer {
		return vscode.NewVSCode()
	},
	"opencode": func() agents.Initializer {
		return opencode.NewOpenCode()
	},
}

func init() {
	subcommand.RegisterExtension("agent", registerAgentCommand)
}

func registerAgentCommand(gs *state.GlobalState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage AI agent integrations for k6",
		Long: `Manage AI agent configurations for supported platforms.

Use "k6 x agent init [platform]" to scaffold one platform or
"k6 x agent init --all" to install all configurations at once.`,
	}

	cmd.AddCommand(newInitCommand(gs))
	cmd.AddCommand(newStatusCommand(gs))

	return cmd
}

func newInitCommand(gs *state.GlobalState) *cobra.Command {
	return newInitCommandWithFactories(gs, defaultInitializerFactories)
}

func newInitCommandWithFactories(gs *state.GlobalState, factories map[string]initializerFactory) *cobra.Command {
	var force bool
	var all bool

	cmd := &cobra.Command{
		Use:   "init [platform]",
		Short: "Initialize AI agent configuration",
		Long: `Initialize AI agent configurations for supported platforms.

Supported platforms:
  - claude: Claude Code agents (.claude folder)
  - vscode: VSCode/Copilot agents (.github/agents and .vscode/mcp.json)
  - opencode: OpenCode prompts (.opencode/prompts and opencode.json)

Examples:
  k6 x agent init claude
  k6 x agent init vscode --force
  k6 x agent init opencode
  k6 x agent init --all
  k6 x agent init --all --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			platforms, err := resolvePlatforms(args, all)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			opts := agents.InitializerOptions{
				Force: force,
			}

			platformList := formatPlatformList(platforms)
			if _, err := fmt.Fprintf(gs.Stdout, "🎯 Initializing k6 agents for %s\n\n", platformList); err != nil {
				return fmt.Errorf("failed to write header: %w", err)
			}

			if factories == nil {
				factories = defaultInitializerFactories
			}

			results := make(map[string]*agents.InitializeResult)
			var initErrors []error

			for _, p := range platforms {
				factory, ok := factories[p]
				if !ok {
					initErrors = append(initErrors, fmt.Errorf("unsupported platform: %s", p))
					continue
				}

				initializer := factory()

				if err := initializer.Validate(ctx, opts); err != nil {
					initErrors = append(initErrors, fmt.Errorf("%s: %w", p, err))
					continue
				}

				result, err := initializer.Initialize(ctx, opts)
				if err != nil {
					initErrors = append(initErrors, fmt.Errorf("%s: %w", p, err))
					continue
				}

				results[p] = result
			}

			if len(initErrors) > 0 {
				for _, err := range initErrors {
					if _, printErr := fmt.Fprintf(gs.Stderr, "❌ Error: %v\n", err); printErr != nil {
						return printErr
					}
				}
				return fmt.Errorf("failed to initialize one or more platforms")
			}

			if err := printResults(gs, platforms, results); err != nil {
				return err
			}

			if _, err := fmt.Fprintf(gs.Stdout, "\n✅ Done.\n"); err != nil {
				return fmt.Errorf("failed to write success message: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing agent folders")
	cmd.Flags().BoolVar(&all, "all", false, "Initialize all supported agent platforms")

	return cmd
}

func newStatusCommand(gs *state.GlobalState) *cobra.Command {
	return newStatusCommandWithReporter(gs, newStatusReporter())
}

func newStatusCommandWithReporter(gs *state.GlobalState, reporter statusCollector) *cobra.Command {
	if reporter == nil {
		reporter = newStatusReporter()
	}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report agent configuration status",
		Long: `Report which agent platforms are initialized in the current workspace
and whether the mcp-k6 dependency is installed locally.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			getwd := os.Getwd
			if gs != nil && gs.Getwd != nil {
				getwd = gs.Getwd
			}

			workingDir, err := getwd()
			if err != nil {
				return fmt.Errorf("failed to resolve working directory: %w", err)
			}

			report, err := reporter.Collect(workingDir)
			if err != nil {
				return fmt.Errorf("failed to collect agent status: %w", err)
			}

			return renderStatusReport(gs, report)
		},
	}

	return cmd
}

func resolvePlatforms(args []string, all bool) ([]string, error) {
	if all {
		if len(args) > 0 {
			return nil, fmt.Errorf("cannot specify a platform argument when using --all; omit the argument to initialize every platform")
		}
		return append([]string(nil), supportedPlatforms...), nil
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("missing platform argument: provide one of %s or use --all", strings.Join(supportedPlatforms, ", "))
	}

	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments: agent init accepts a single platform")
	}

	provided := strings.TrimSpace(args[0])
	platform := strings.ToLower(provided)
	switch platform {
	case "claude":
		return []string{"claude"}, nil
	case "vscode":
		return []string{"vscode"}, nil
	case "opencode":
		return []string{"opencode"}, nil
	default:
		return nil, fmt.Errorf("invalid platform %q: must be one of %s", provided, strings.Join(supportedPlatforms, ", "))
	}
}

// formatPlatformList formats a list of platform names for display.
func formatPlatformList(platforms []string) string {
	if len(platforms) == 0 {
		return ""
	}
	if len(platforms) == 1 {
		return platformDisplayName(platforms[0])
	}

	names := make([]string, len(platforms))
	for i, p := range platforms {
		names[i] = platformDisplayName(p)
	}
	return strings.Join(names, " and ")
}

// platformDisplayName returns the display name for a platform.
func platformDisplayName(platform string) string {
	switch platform {
	case "claude":
		return "Claude Code"
	case "vscode":
		return "VSCode/GitHub Copilot"
	case "opencode":
		return "OpenCode"
	default:
		return platform
	}
}

// printResults prints the initialization results for all platforms.
func printResults(gs *state.GlobalState, platforms []string, results map[string]*agents.InitializeResult) error {
	for i, p := range platforms {
		result, ok := results[p]
		if !ok {
			continue
		}

		// Add spacing between platforms if there are multiple
		if i > 0 {
			if _, err := fmt.Fprintln(gs.Stdout); err != nil {
				return err
			}
		}

		// Print platform-specific results
		switch p {
		case "claude":
			if err := printClaudeResults(gs, result); err != nil {
				return err
			}
		case "vscode":
			if err := printVSCodeResults(gs, result); err != nil {
				return err
			}
		case "opencode":
			if err := printOpenCodeResults(gs, result); err != nil {
				return err
			}
		}
	}
	return nil
}

// printClaudeResults prints the results for Claude Code initialization.
func printClaudeResults(gs *state.GlobalState, result *agents.InitializeResult) error {
	if len(result.FilesCreated) == 0 {
		return nil
	}

	// Count agent files
	agentCount := countFilesMatching(result.FilesCreated, ".claude/agents/")

	// Print grouped summary
	if _, err := fmt.Fprintln(gs.Stdout, "📁 Claude Code"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(gs.Stdout, "   Created .claude/ with %d agents and MCP settings\n", agentCount); err != nil {
		return err
	}

	return nil
}

// printVSCodeResults prints the results for VSCode initialization.
func printVSCodeResults(gs *state.GlobalState, result *agents.InitializeResult) error {
	if len(result.FilesCreated) == 0 {
		return nil
	}

	// Count agent files
	agentCount := countFilesMatching(result.FilesCreated, ".github/agents/")

	// Print grouped summary
	if _, err := fmt.Fprintln(gs.Stdout, "📁 VSCode/GitHub Copilot"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(gs.Stdout, "   Created .github/agents/ with %d agents\n", agentCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(gs.Stdout, "   Created .vscode/mcp.json with local MCP settings"); err != nil {
		return err
	}

	// Print numbered steps for GitHub Copilot MCP configuration
	if _, err := fmt.Fprintln(gs.Stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(gs.Stdout, "⚠️  To enable MCP in GitHub Copilot:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(gs.Stdout, "   1. Go to github.com > Settings > Copilot > Coding agent"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(gs.Stdout, "   2. Scroll to \"MCP configuration\" and add the k6 server:"); err != nil {
		return err
	}

	// Create the MCP configuration JSON with indentation for display
	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"k6": map[string]interface{}{
				"type":    "stdio",
				"command": "mcp-k6",
				"args":    []string{},
				"tools":   []string{"*"},
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(mcpConfig, "      ", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MCP configuration: %w", err)
	}

	if _, err := fmt.Fprintln(gs.Stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(gs.Stdout, "      %s\n", string(jsonBytes)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(gs.Stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(gs.Stdout, "   3. Save your changes"); err != nil {
		return err
	}

	return nil
}

// printOpenCodeResults prints the results for OpenCode initialization.
func printOpenCodeResults(gs *state.GlobalState, result *agents.InitializeResult) error {
	if len(result.FilesCreated) == 0 {
		return nil
	}

	// Count prompt files
	promptCount := countFilesMatching(result.FilesCreated, ".opencode/prompts/")

	// Print grouped summary
	if _, err := fmt.Fprintln(gs.Stdout, "📁 OpenCode"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(gs.Stdout, "   Created .opencode/prompts/ with %d prompts\n", promptCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(gs.Stdout, "   Created opencode.json with MCP and agent settings"); err != nil {
		return err
	}

	return nil
}

// countFilesMatching counts files that contain the given pattern.
func countFilesMatching(files []string, pattern string) int {
	count := 0
	for _, file := range files {
		if strings.Contains(file, pattern) {
			count++
		}
	}
	return count
}

func renderStatusReport(gs *state.GlobalState, report *StatusReport) error {
	if gs == nil || gs.Stdout == nil {
		return fmt.Errorf("invalid global state: stdout is not available")
	}

	if _, err := fmt.Fprintln(gs.Stdout, "🤖 Agent installation status"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(gs.Stdout); err != nil {
		return err
	}

	for i, platform := range report.Platforms {
		if err := printPlatformStatus(gs, platform); err != nil {
			return err
		}
		if i < len(report.Platforms)-1 {
			if _, err := fmt.Fprintln(gs.Stdout); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintln(gs.Stdout); err != nil {
		return err
	}
	return printMCPStatus(gs, report.MCP)
}

func printPlatformStatus(gs *state.GlobalState, status PlatformStatus) error {
	icon := "❌"
	if status.Installed {
		icon = "✅"
	}

	if _, err := fmt.Fprintf(gs.Stdout, "%s %s\n", icon, platformDisplayName(status.Name)); err != nil {
		return err
	}

	for _, detail := range status.Details {
		if _, err := fmt.Fprintf(gs.Stdout, "   - %s\n", detail); err != nil {
			return err
		}
	}

	if !status.Installed {
		if len(status.Missing) == 0 {
			if _, err := fmt.Fprintf(gs.Stdout, "   - Not detected in this workspace\n"); err != nil {
				return err
			}
		} else {
			for _, missing := range status.Missing {
				if _, err := fmt.Fprintf(gs.Stdout, "   - Missing: %s\n", missing); err != nil {
					return err
				}
			}
		}

		if _, err := fmt.Fprintf(gs.Stdout, "   - Hint: k6 x agent init %s\n", status.Name); err != nil {
			return err
		}
	}

	return nil
}

func printMCPStatus(gs *state.GlobalState, status MCPStatus) error {
	icon := "✅"
	if !status.Installed {
		icon = "❌"
	}

	if _, err := fmt.Fprintf(gs.Stdout, "%s mcp-k6 dependency\n", icon); err != nil {
		return err
	}

	if status.Installed {
		if _, err := fmt.Fprintf(gs.Stdout, "   - Found at %s\n", status.Path); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(gs.Stdout, "   - Not found on PATH\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(gs.Stdout, "   - Install instructions: https://github.com/grafana/mcp-k6\n"); err != nil {
			return err
		}
	}

	return nil
}
