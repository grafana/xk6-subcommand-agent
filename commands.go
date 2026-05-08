// Package initagents provides k6 subcommand for initializing AI agent configurations.
package initagents

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"go.k6.io/k6/v2/cmd/state"

	"github.com/grafana/xk6-subcommand-agent/agents"
	"github.com/grafana/xk6-subcommand-agent/agents/adapters"
	"github.com/grafana/xk6-subcommand-agent/agents/core"
)

func newListCommand(gs *state.GlobalState) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available targets and skills",
		RunE: func(_ *cobra.Command, _ []string) error {
			if _, err := fmt.Fprintln(gs.Stdout, "Targets:"); err != nil {
				return err
			}

			for _, t := range adapters.All() {
				if _, err := fmt.Fprintf(gs.Stdout, "  %-20s %s\n", t.Name(), t.DisplayName()); err != nil {
					return err
				}
			}

			skills, err := core.LoadSkills(agents.SkillsFS)
			if err != nil {
				return fmt.Errorf("failed to load skills: %w", err)
			}

			if _, err := fmt.Fprintf(gs.Stdout, "\nSkills (%d):\n", len(skills)); err != nil {
				return err
			}

			tw := tabwriter.NewWriter(gs.Stdout, 0, 0, 2, ' ', 0)
			for _, s := range skills {
				desc := truncate(s.Description, 72)
				if _, err := fmt.Fprintf(tw, "  %s\t%s\n", s.Name, desc); err != nil {
					return err
				}
			}

			return tw.Flush()
		},
	}
}

func newSkillsCommand(gs *state.GlobalState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage skills shipped in the binary",
	}

	cmd.AddCommand(newSkillsListCommand(gs))
	cmd.AddCommand(newSkillsShowCommand(gs))

	return cmd
}

func newSkillsListCommand(gs *state.GlobalState) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List skills shipped in the binary",
		RunE: func(_ *cobra.Command, _ []string) error {
			skills, err := core.LoadSkills(agents.SkillsFS)
			if err != nil {
				return fmt.Errorf("failed to load skills: %w", err)
			}

			tw := tabwriter.NewWriter(gs.Stdout, 0, 0, 2, ' ', 0)
			if _, err := fmt.Fprintf(tw, "NAME\tDESCRIPTION\n"); err != nil {
				return err
			}

			for _, s := range skills {
				desc := truncate(s.Description, 72)
				if _, err := fmt.Fprintf(tw, "%s\t%s\n", s.Name, desc); err != nil {
					return err
				}
			}

			return tw.Flush()
		},
	}
}

func newSkillsShowCommand(gs *state.GlobalState) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Print a skill's SKILL.md to stdout",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			skills, err := core.LoadSkills(agents.SkillsFS)
			if err != nil {
				return fmt.Errorf("failed to load skills: %w", err)
			}

			name := args[0]

			for _, s := range skills {
				if s.Name == name {
					_, err := fmt.Fprint(gs.Stdout, s.Body)
					return err
				}
			}

			return fmt.Errorf("skill %q not found", name)
		},
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
