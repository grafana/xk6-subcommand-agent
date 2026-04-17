# xk6-agent

xk6-agent is a [k6](https://k6.io/) extension that adds the `agent` subcommand under `k6 x`.  
With a single `k6 x agent` invocation, you bootstrap Model Context Protocol (MCP) agent
configurations for your k6 workspaces so that Claude Code, GitHub Copilot/VS Code, and
OpenCode can all talk to the local `mcp-k6` server for documentation lookup, script
validation, and k6 test execution. Most importantly, these agents ship dedicated sub-agents
and prompts for **generating k6 scripts from requirements** and **converting Playwright tests
into k6 browser scripts**, so you can go from specs or existing tests to runnable k6 code in
one guided workflow.

## Highlights

- One command to scaffold Claude Code, VSCode/GitHub Copilot, or OpenCode agent bundles tailored to k6.
- Batteries-included prompts for both the **k6 test generator** and **Playwright-to-k6 converter** workflows.
- Safe re-runs via `--force`, so you can refresh agent definitions whenever templates change.
- Built-in `k6 x agent status` to confirm what is installed and whether `mcp-k6` is reachable on your PATH.

## Prerequisites

- Go 1.25.5 or newer.
- [`xk6`](https://github.com/grafana/xk6) to build custom k6 binaries.
- [`mcp-k6`](https://github.com/grafana/mcp-k6) (must be on your PATH so the generated configs can find it).
- An MCP-capable AI client for the platform you plan to initialize (Claude Desktop, GitHub Copilot with MCP beta, or OpenCode).

## Installation

### Build a k6 binary with the agent extension

Install `xk6` and build a custom k6 binary that includes this extension. Once built, the
binary exposes the `k6 x agent` command group.

```bash
go install go.k6.io/xk6/cmd/xk6@latest
xk6 build v0.52.0 --with github.com/grafana/xk6-agent@latest
mv k6 ~/bin/k6-agent   # optional: keep a dedicated binary
~/bin/k6-agent version
```

> Use the k6 version number that matches your environment. Every time you pull new changes
> from this repo you should rebuild the binary so the CLI picks them up.

## Getting started

Run the commands below inside the workspace that should receive the agent files (usually the
root of your k6 project). xk6-agent will create platform-specific configuration under
`.claude/`, `.github/agents`, `.vscode/`, and `.opencode/`.

### Initialize agent configs

Use the `init` subcommand to scaffold MCP-aware agents for the platforms you care about:

```bash
k6 x agent init claude
k6 x agent init vscode --force
k6 x agent init claude opencode   # install multiple platforms in one go
k6 x agent init --all             # claude + vscode + opencode
```

- `--force` removes existing agent folders before recreating them.
- `--all` installs every supported platform; do not pass a positional platform name when using `--all`.

### Check installation status

Use the `status` subcommand to see what has been installed in the current workspace and
whether the `mcp-k6` server is available on your PATH:

```bash
k6 x agent status
```

Example output:

```
🤖 Agent installation status

✅ Claude Code
   - 2 agent file(s) in .claude/agents

❌ VSCode/GitHub Copilot
   - Missing: .github/agents directory
   - Hint: k6 x agent init vscode

❌ OpenCode
   - Missing: .opencode/prompts directory
   - Hint: k6 x agent init opencode

❌ mcp-k6 dependency
   - Not found on PATH
   - Install instructions: https://github.com/grafana/mcp-k6
```

Once `k6 x agent init ...` and `k6 x agent status` both succeed, your MCP-capable editor can
start using the generated sub-agents to call into `mcp-k6`.

## Agent workflows

xk6-agent focuses on making two core MCP workflows easy to use from your editor of choice:

- **k6 test generator** – generate and iterate on k6 scripts with the help of `mcp-k6`
  documentation, validation, and execution tools.
- **Playwright-to-k6 converter** – convert existing Playwright tests into k6 scripts, then
  refine and validate them through the same MCP tools.

Each platform gets agents or prompts that are tailored to these workflows but share the same
underlying templates.

## Artifacts and conventions

The `k6 x agent` commands generate a simple, auditable structure in your k6 project:

### Generated files

| Platform | Files & directories | Purpose |
| --- | --- | --- |
| Claude Code | `.claude/`, `.claude/agents/k6-test-generator.md`, `.claude/agents/k6-playwright-test-converter.md`, `.claude/settings.local.json` | Adds two Claude sub-agents and enables the local MCP server. |
| VSCode / GitHub Copilot | `.github/agents/k6-test-generator.agent.md`, `.github/agents/k6-playwright-test-converter.agent.md`, `.vscode/mcp.json` | Ships GitHub Copilot agents plus an MCP configuration pointing at `mcp-k6`. |
| OpenCode | `.opencode/`, `.opencode/prompts/*.md`, `opencode.json` | Creates OpenCode sub-agent prompts and registers `mcp-k6` as a local server. |

The generated Markdown files contain platform-specific frontmatter plus a shared body sourced
from `agents/templates/*.tmpl.md` inside this repository.

### MCP server dependency

All platforms expect a local MCP server named `k6` that executes the `mcp-k6` binary. Install
it and make sure it is on your PATH:

```bash
brew install grafana/tap/mcp-k6      # or download from the GitHub releases page
which mcp-k6
```

Re-run `k6 x agent status` after installation to verify the dependency shows up as ✅.

## Customizing prompts and templates

- Edit the generated Markdown files inside `.claude/agents`, `.github/agents`, or
  `.opencode/prompts` if you only need workspace-level tweaks.
- To change the shared templates, update the files under `agents/templates/` and rebuild your
  k6 binary; future `k6 x agent init` runs will pick up the new wording automatically.
- Template frontmatter formatters live in `agents/claude/formatter.go`, `agents/vscode/formatter.go`,
  and `agents/opencode/formatter.go` if you need deeper changes.

## Development

```bash
git clone https://github.com/grafana/xk6-agent.git
cd xk6-agent
go test ./...
golangci-lint run          # optional, requires golangci-lint >= 1.60
```

Tips:

- Go 1.25.5+ is required (matching `go.mod`).
- Tests cover both the shared `agents` package and platform-specific logic (`agents/*.go` plus sub-packages).
- Use `go test ./agents/... -run TestName` to iterate on a single initializer.
- When adding new templates, reference them in `agents/template.go` and keep file permissions explicit so scaffolding works on all platforms.

## Project layout

- `register.go` – wires the extension into the `k6 x agent` command group.
- `status.go` – collects filesystem + `mcp-k6` status and renders human-readable output.
- `agents/` – shared interfaces, structs, template loader, and helper utilities.
- `agents/claude`, `agents/vscode`, `agents/opencode` – platform-specific initializers.
- `agents/templates/` – embedded Markdown/Frontmatter templates used for every agent.
- `scripts/` – sample helper scripts used when refining prompt content.
- `.github/agents` – example GitHub Copilot agent definitions useful for manual testing.

## Troubleshooting

- **“folder already exists”** – rerun with `--force` to delete and recreate platform folders.
- **`mcp-k6` not found** – install it and ensure your shell PATH matches the environment launching Claude, VS Code, or OpenCode.
- **Permission errors** – run the CLI from a writable workspace root and avoid network-mounted folders that block `chmod`.
- **Custom k6 binary not used** – double-check that the `k6` on your PATH is the xk6-built binary containing this extension (`which k6`).

## Support

Please open an issue or PR on this repository if you run into bugs, have suggestions for new
agent types, or want to contribute additional templates.

## License

Licensed under the Apache 2.0 License. See `LICENSE` for the full text once it has been added
to the repository.


