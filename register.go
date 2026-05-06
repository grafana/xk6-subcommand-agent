// Package initagents provides k6 subcommand for initializing AI agent configurations.
//
//nolint:forbidigo
package initagents

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.k6.io/k6/v2/cmd/state"
	"go.k6.io/k6/v2/subcommand"

	"github.com/grafana/xk6-agent/agents"
	"github.com/grafana/xk6-agent/agents/adapters"
	"github.com/grafana/xk6-agent/agents/core"
	"github.com/grafana/xk6-agent/agents/mcp"

	// Register all adapter targets via init().
	_ "github.com/grafana/xk6-agent/agents/adapters/claude_code"
	_ "github.com/grafana/xk6-agent/agents/adapters/cline"
	_ "github.com/grafana/xk6-agent/agents/adapters/codex_cli"
	_ "github.com/grafana/xk6-agent/agents/adapters/cursor"
	_ "github.com/grafana/xk6-agent/agents/adapters/opencode"
	_ "github.com/grafana/xk6-agent/agents/adapters/vscode_copilot"
)

func init() {
	subcommand.RegisterExtension("agent", registerAgentCommand)
}

func registerAgentCommand(gs *state.GlobalState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage AI agent integrations for k6",
		Long: `Manage AI agent configurations for supported platforms.

Use "k6 x agent init [target]" to scaffold one platform or
"k6 x agent init --all" to install all configurations at once.`,
	}

	cmd.AddCommand(newInitCommand(gs))
	cmd.AddCommand(newStatusCommand(gs))
	cmd.AddCommand(newListCommand(gs))
	cmd.AddCommand(newSkillsCommand(gs))

	return cmd
}

func newInitCommand(gs *state.GlobalState) *cobra.Command {
	var force bool
	var all bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "init [target] [target...]",
		Short: "Initialize AI agent configuration",
		Long: fmt.Sprintf(`Initialize AI agent configurations for supported targets.

Supported targets:
%s

Examples:
  k6 x agent init claude-code
  k6 x agent init vscode-copilot --force
  k6 x agent init opencode
  k6 x agent init --all
  k6 x agent init --all --force
  k6 x agent init claude-code vscode-copilot`, formatTargetList()),
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := resolveTargets(args, all)
			if err != nil {
				return err
			}

			return runInit(cmd.Context(), gs, targets, force, dryRun)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing agent folders")
	cmd.Flags().BoolVar(&all, "all", false, "Initialize all supported agent platforms")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print plan without writing files")

	return cmd
}

func runInit(ctx context.Context, gs *state.GlobalState, targets []adapters.Target, force, dryRun bool) error {
	// Load skills and MCP config.
	skills, err := core.LoadSkills(agents.SkillsFS)
	if err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	mcpCfg, err := core.LoadMCPConfig(mcp.ServersYAML)
	if err != nil {
		return fmt.Errorf("failed to load MCP config: %w", err)
	}

	// Resolve project root.
	root, err := resolveRoot(gs)
	if err != nil {
		return err
	}

	displayNames := make([]string, len(targets))
	for i, t := range targets {
		displayNames[i] = t.DisplayName()
	}

	header := fmt.Sprintf("Initializing k6 agents for %s\n\n", strings.Join(displayNames, " and "))
	if _, err := fmt.Fprint(gs.Stdout, header); err != nil {
		return err
	}

	in := adapters.Inputs{
		Skills: skills,
		MCP:    mcpCfg,
		Root:   root,
	}

	var initErrors []error

	for _, t := range targets {
		if err := initTarget(ctx, gs, t, in, force, dryRun); err != nil {
			initErrors = append(initErrors, err)
		}
	}

	if len(initErrors) > 0 {
		for _, err := range initErrors {
			if _, printErr := fmt.Fprintf(gs.Stderr, "Error: %v\n", err); printErr != nil {
				return printErr
			}
		}

		return fmt.Errorf("failed to initialize one or more targets")
	}

	if !dryRun {
		if _, err := fmt.Fprintf(gs.Stdout, "\nDone.\n"); err != nil {
			return err
		}
	}

	return nil
}

func initTarget(
	ctx context.Context,
	gs *state.GlobalState,
	t adapters.Target,
	in adapters.Inputs,
	force, dryRun bool,
) error {
	plan, err := t.Plan(ctx, in)
	if err != nil {
		return fmt.Errorf("%s: %w", t.Name(), err)
	}

	if dryRun {
		return printDryRun(gs, t, plan)
	}

	coreFiles := adaptPlanFiles(plan)
	outcomes, err := core.Apply(coreFiles, in.Root, core.ApplyOptions{Force: force})
	if err != nil {
		return fmt.Errorf("%s: %w", t.Name(), err)
	}

	if err := printOutcomes(gs, t, outcomes); err != nil {
		return err
	}

	for _, notice := range plan.Notices {
		if _, err := fmt.Fprintf(gs.Stdout, "   %s\n", notice); err != nil {
			return err
		}
	}

	return nil
}

// resolveRoot returns the project root directory.
func resolveRoot(gs *state.GlobalState) (string, error) {
	getwd := os.Getwd
	if gs != nil && gs.Getwd != nil {
		getwd = gs.Getwd
	}

	root, err := getwd()
	if err != nil {
		return "", fmt.Errorf("failed to resolve working directory: %w", err)
	}

	return root, nil
}

// resolveTargets resolves CLI args into registered Target instances.
func resolveTargets(args []string, all bool) ([]adapters.Target, error) {
	if all {
		if len(args) > 0 {
			return nil, fmt.Errorf("cannot specify target arguments when using --all")
		}

		return adapters.All(), nil
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("missing target argument: provide one of %s or use --all",
			strings.Join(adapters.Names(), ", "))
	}

	targets := make([]adapters.Target, 0, len(args))

	for _, arg := range args {
		name := strings.ToLower(strings.TrimSpace(arg))
		t, ok := adapters.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown target %q: must be one of %s",
				arg, strings.Join(adapters.Names(), ", "))
		}

		targets = append(targets, t)
	}

	return targets, nil
}

// formatTargetList returns a formatted list of available targets for help text.
func formatTargetList() string {
	var sb strings.Builder
	for _, t := range adapters.All() {
		fmt.Fprintf(&sb, "  - %s: %s\n", t.Name(), t.DisplayName())
	}

	return sb.String()
}

// adaptPlanFiles converts adapter PlannedFiles to core PlannedFiles.
func adaptPlanFiles(plan adapters.Plan) []core.PlannedFile {
	files := make([]core.PlannedFile, len(plan.Files))
	for i, f := range plan.Files {
		files[i] = core.PlannedFile{
			Path:        f.Path,
			Content:     f.Content,
			Mode:        core.WriteMode(f.Mode),
			MergeKey:    f.MergeKey,
			OwnerMarker: f.OwnerMarker,
		}
	}

	return files
}

// printDryRun prints the plan for a target without executing it.
func printDryRun(gs *state.GlobalState, t adapters.Target, plan adapters.Plan) error {
	if _, err := fmt.Fprintf(gs.Stdout, "%s (dry-run):\n", t.DisplayName()); err != nil {
		return err
	}

	for _, f := range plan.Files {
		mode := "create"
		switch f.Mode {
		case adapters.CreateOnly:
			mode = "create"
		case adapters.MergeJSONByKey:
			mode = "merge"
		case adapters.OverwriteIfManaged:
			mode = "update"
		}

		if _, err := fmt.Fprintf(gs.Stdout, "  [%s] %s (%d bytes)\n", mode, f.Path, len(f.Content)); err != nil {
			return err
		}
	}

	for _, n := range plan.Notices {
		if _, err := fmt.Fprintf(gs.Stdout, "  note: %s\n", n); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(gs.Stdout)

	return err
}

// printOutcomes prints the results of applying a plan.
func printOutcomes(gs *state.GlobalState, t adapters.Target, outcomes []core.Outcome) error {
	created := 0
	updated := 0

	for _, o := range outcomes {
		switch o.Status {
		case core.Created:
			created++
		case core.Updated:
			updated++
		case core.Skipped, core.Warned, core.Errored:
			// no-op for display purposes
		}
	}

	if _, err := fmt.Fprintf(gs.Stdout, "%s\n", t.DisplayName()); err != nil {
		return err
	}

	if created > 0 {
		if _, err := fmt.Fprintf(gs.Stdout, "   Created %d file(s)\n", created); err != nil {
			return err
		}
	}

	if updated > 0 {
		if _, err := fmt.Fprintf(gs.Stdout, "   Updated %d file(s)\n", updated); err != nil {
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

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report agent configuration status",
		Long: `Report which agent platforms are initialized in the current workspace
and whether k6 with MCP support is installed locally.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := resolveRoot(gs)
			if err != nil {
				return err
			}

			report, err := reporter.Collect(root)
			if err != nil {
				return fmt.Errorf("failed to collect agent status: %w", err)
			}

			return renderStatusReport(gs, report)
		},
	}

	return cmd
}

func renderStatusReport(gs *state.GlobalState, report *StatusReport) error {
	if gs == nil || gs.Stdout == nil {
		return fmt.Errorf("invalid global state: stdout is not available")
	}

	if _, err := fmt.Fprintln(gs.Stdout, "Agent installation status"); err != nil {
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
	icon := "[-]"
	if status.Installed {
		icon = "[+]"
	}

	t, ok := adapters.Get(status.Name)
	displayName := status.Name
	if ok {
		displayName = t.DisplayName()
	}

	if _, err := fmt.Fprintf(gs.Stdout, "%s %s\n", icon, displayName); err != nil {
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

	if err := printMissingDetails(gs, status); err != nil {
		return err
	}

	_, err := fmt.Fprintf(gs.Stdout, "   - Hint: k6 x agent init %s\n", status.Name)

	return err
}

func printMissingDetails(gs *state.GlobalState, status PlatformStatus) error {
	if len(status.Missing) == 0 {
		_, err := fmt.Fprintf(gs.Stdout, "   - Not detected in this workspace\n")
		return err
	}

	for _, missing := range status.Missing {
		if _, err := fmt.Fprintf(gs.Stdout, "   - Missing: %s\n", missing); err != nil {
			return err
		}
	}

	return nil
}

func printMCPStatus(gs *state.GlobalState, status MCPStatus) error {
	icon := "[+]"
	if !status.Installed {
		icon = "[-]"
	}

	if _, err := fmt.Fprintf(gs.Stdout, "%s k6 MCP support\n", icon); err != nil {
		return err
	}

	if status.Installed {
		if _, err := fmt.Fprintf(gs.Stdout, "   - Found at %s\n", status.Path); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(gs.Stdout, "   - k6 not found on PATH\n"); err != nil {
			return err
		}
		installURL := "https://grafana.com/docs/k6/latest/set-up/install-k6/"
		if _, err := fmt.Fprintf(gs.Stdout, "   - Install k6: %s\n", installURL); err != nil {
			return err
		}
	}

	return nil
}
