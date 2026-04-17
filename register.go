// Package initagents wires the `agent` command group into k6's `x`
// subcommand surface. All real behavior lives in the data-driven
// [github.com/grafana/xk6-agent/agents] package.
package initagents

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"go.k6.io/k6/cmd/state"
	"go.k6.io/k6/subcommand"

	"github.com/grafana/xk6-agent/agents"
)

func init() {
	subcommand.RegisterExtension("agent", registerAgentCommand)
}

func registerAgentCommand(gs *state.GlobalState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage AI agent integrations for k6",
		Long: `Manage AI agent configurations for supported platforms.

Use "k6 x agent init [platforms...]" to scaffold one or more platforms, or
"k6 x agent init --all" to install every supported configuration at once.`,
	}
	cmd.AddCommand(newInitCommand(gs))
	cmd.AddCommand(newStatusCommand(gs))
	return cmd
}

// platformIDs returns every registered platform ID, in declaration order.
func platformIDs() []string {
	ids := make([]string, 0, len(agents.Platforms))
	for _, p := range agents.Platforms {
		ids = append(ids, p.ID)
	}
	return ids
}

// lookupPlatform returns the Platform with the given ID or an error listing
// the valid choices.
func lookupPlatform(id string) (agents.Platform, error) {
	for _, p := range agents.Platforms {
		if p.ID == id {
			return p, nil
		}
	}
	return agents.Platform{}, fmt.Errorf(
		"invalid platform %q: must be one of %s",
		id, strings.Join(platformIDs(), ", "),
	)
}

// resolvePlatforms turns positional args + --all into the list of Platforms
// to install. Duplicate args are collapsed, order is preserved.
func resolvePlatforms(args []string, all bool) ([]agents.Platform, error) {
	if all {
		if len(args) > 0 {
			return nil, fmt.Errorf(
				"cannot specify a platform argument with --all; omit arguments to install every platform",
			)
		}
		out := make([]agents.Platform, len(agents.Platforms))
		copy(out, agents.Platforms)
		return out, nil
	}
	if len(args) == 0 {
		return nil, fmt.Errorf(
			"missing platform argument: provide one or more of %s, or use --all",
			strings.Join(platformIDs(), ", "),
		)
	}

	seen := make(map[string]bool, len(args))
	out := make([]agents.Platform, 0, len(args))
	for _, raw := range args {
		id := strings.ToLower(strings.TrimSpace(raw))
		if seen[id] {
			continue
		}
		p, err := lookupPlatform(id)
		if err != nil {
			return nil, err
		}
		seen[id] = true
		out = append(out, p)
	}
	return out, nil
}

func newInitCommand(gs *state.GlobalState) *cobra.Command {
	var force, all bool

	cmd := &cobra.Command{
		Use:   "init [platform...]",
		Short: "Initialize AI agent configuration",
		Long: `Initialize AI agent configurations for one or more supported platforms.

Supported platforms: ` + strings.Join(platformIDs(), ", ") + `.

Examples:
  k6 x agent init claude
  k6 x agent init vscode --force
  k6 x agent init claude opencode
  k6 x agent init --all
  k6 x agent init --all --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := resolvePlatforms(args, all)
			if err != nil {
				return err
			}

			opts := agents.InitializerOptions{Force: force}
			fs := &agents.OSFileSystem{}
			ctx := cmd.Context()

			names := make([]string, len(targets))
			for i, t := range targets {
				names[i] = t.DisplayName
			}
			if _, err := fmt.Fprintf(gs.Stdout, "🎯 Initializing k6 agents for %s\n\n", joinList(names)); err != nil {
				return err
			}

			for i, p := range targets {
				result, err := agents.Install(ctx, fs, p, opts)
				if err != nil {
					if _, perr := fmt.Fprintf(gs.Stderr, "❌ %s: %v\n", p.ID, err); perr != nil {
						return perr
					}
					return fmt.Errorf("failed to initialize %s", p.ID)
				}
				if i > 0 {
					if _, err := fmt.Fprintln(gs.Stdout); err != nil {
						return err
					}
				}
				if err := printInstallSummary(gs, p, result); err != nil {
					return err
				}
			}

			if _, err := fmt.Fprintln(gs.Stdout, "\n✅ Done."); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing agent folders")
	cmd.Flags().BoolVar(&all, "all", false, "Initialize all supported agent platforms")
	return cmd
}

// printInstallSummary renders a compact per-platform summary: a header, a
// count of agent files written, the list of other files created, and the
// platform's post-install message if any.
func printInstallSummary(gs *state.GlobalState, p agents.Platform, result *agents.InitializeResult) error {
	if _, err := fmt.Fprintf(gs.Stdout, "📁 %s\n", p.DisplayName); err != nil {
		return err
	}

	// Group: agent files vs. everything else.
	var agentFiles, otherFiles []string
	for _, f := range result.FilesCreated {
		if strings.Contains(f, "/agents/") || strings.Contains(f, "/prompts/") {
			agentFiles = append(agentFiles, f)
		} else {
			otherFiles = append(otherFiles, f)
		}
	}
	if len(agentFiles) > 0 {
		if _, err := fmt.Fprintf(gs.Stdout, "   %d agent file(s)\n", len(agentFiles)); err != nil {
			return err
		}
	}
	sort.Strings(otherFiles)
	for _, f := range otherFiles {
		if _, err := fmt.Fprintf(gs.Stdout, "   created %s\n", f); err != nil {
			return err
		}
	}
	if p.PostInstall != "" {
		if _, err := fmt.Fprintln(gs.Stdout); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(gs.Stdout, p.PostInstall); err != nil {
			return err
		}
	}
	return nil
}

func newStatusCommand(gs *state.GlobalState) *cobra.Command {
	return newStatusCommandWithReporter(gs, newStatusReporter())
}

func newStatusCommandWithReporter(gs *state.GlobalState, reporter statusCollector) *cobra.Command {
	if reporter == nil {
		reporter = newStatusReporter()
	}
	return &cobra.Command{
		Use:   "status",
		Short: "Report agent configuration status",
		Long: `Report which agent platforms are initialized in the current workspace
and whether the mcp-k6 dependency is installed locally.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			getwd := os.Getwd //nolint:forbidigo // fallback when GlobalState.Getwd is nil
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
}

// joinList formats a list of names like "A and B" or "A, B, and C".
func joinList(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
	}
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
	if _, err := fmt.Fprintf(gs.Stdout, "%s %s\n", icon, status.DisplayName); err != nil {
		return err
	}
	for _, detail := range status.Details {
		if _, err := fmt.Fprintf(gs.Stdout, "   - %s\n", detail); err != nil {
			return err
		}
	}
	if status.Installed {
		return nil
	}
	return printNotInstalledDetails(gs, status)
}

func printNotInstalledDetails(gs *state.GlobalState, status PlatformStatus) error {
	if len(status.Missing) == 0 {
		if _, err := fmt.Fprintln(gs.Stdout, "   - Not detected in this workspace"); err != nil {
			return err
		}
	}
	for _, missing := range status.Missing {
		if _, err := fmt.Fprintf(gs.Stdout, "   - Missing: %s\n", missing); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(gs.Stdout, "   - Hint: k6 x agent init %s\n", status.ID)
	return err
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
		_, err := fmt.Fprintf(gs.Stdout, "   - Found at %s\n", status.Path)
		return err
	}
	if _, err := fmt.Fprintf(gs.Stdout, "   - Not found on PATH\n"); err != nil {
		return err
	}
	_, err := fmt.Fprintf(gs.Stdout, "   - Install instructions: https://github.com/grafana/mcp-k6\n")
	return err
}
