# xk6-agent

> Bootstrap an AI-assisted k6 testing workflow in any editor — Claude Code, Cursor, Copilot, Codex CLI, OpenCode, or Cline.

`xk6-agent` powers the `k6 x agent` command group, shipped as part of [k6](https://k6.io/). One invocation drops a set of portable [`SKILL.md`](https://docs.claude.com/en/docs/agents/skills) bundles into your project and wires every supported AI coding tool to the built-in `k6 x mcp` server. The bundled skills cover planning, smoke / load / browser tests, and Playwright-to-k6 conversion.

## Contents

- [Quickstart](#quickstart)
- [Supported targets](#supported-targets)
- [Bundled skills](#bundled-skills)
- [Commands](#commands)
- [How it works](#how-it-works)
- [Safety](#safety)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [License](#license)

## Quickstart

Make sure you have a recent [k6](https://grafana.com/docs/k6/latest/set-up/install-k6/) on your `PATH`, then from the root of your k6 project:

```bash
# Scaffold one or more editor targets.
k6 x agent init claude-code           # one editor
k6 x agent init claude-code cursor    # several
k6 x agent init --all                 # every supported target

# Verify what was installed.
k6 x agent status
```

Open your editor and ask it to plan or write a k6 test — the bundled skills auto-activate based on intent ("write a smoke test for…", "convert this Playwright script…").

## Supported targets

| Target           | CLI name         | Skills location              | MCP config                     |
| ---------------- | ---------------- | ---------------------------- | ------------------------------ |
| Claude Code      | `claude-code`    | `.claude/skills/<name>/`     | `.mcp.json` + `.claude/settings.local.json` |
| OpenCode         | `opencode`       | `.opencode/skills/<name>/`   | `opencode.json`                |
| OpenAI Codex CLI | `codex-cli`      | `.codex/skills/<name>/`      | `.codex/mcp.json`              |
| GitHub Copilot   | `vscode-copilot` | `.github/copilot/skills/<name>/` | `.vscode/mcp.json`         |
| Cursor           | `cursor`         | `.cursor/rules/<name>.mdc`   | `.cursor/mcp.json`             |
| Cline            | `cline`          | `.clinerules/<name>.md`      | global — printed as a notice [^cline] |

[^cline]: Cline's MCP config is global, not project-scoped. `init cline` prints the JSON snippet to add to `cline_mcp_settings.json`.

## Bundled skills

Skills live in `agents/skills/` and are embedded into the binary.

| Skill                     | Triggers on                                                              |
| ------------------------- | ------------------------------------------------------------------------ |
| `k6-test-planner`         | "plan tests", "design a test strategy", "what k6 tests should I write"   |
| `k6-load-test`            | "write a k6 script", "load test", "stress / soak / spike test"           |
| `k6-smoke-test`           | "smoke test", "sanity check", "quick health check"                       |
| `k6-browser-test`         | "browser test", "UI test with k6", "test the frontend"                   |
| `k6-playwright-converter` | A Playwright script that needs to be ported to `k6/browser`              |

Run `k6 x agent skills show <name>` to print a skill's full `SKILL.md`.

## Commands

```
k6 x agent init <target>...   Write or update files for one or more targets
k6 x agent init --all         Initialize every registered target
k6 x agent status             Show what is installed in the current workspace
k6 x agent list               List available targets and bundled skills
k6 x agent skills list        List skills shipped in the binary
k6 x agent skills show <name> Print a skill's SKILL.md to stdout
```

Common flags for `init`:

- `--dry-run` — print the plan (paths, write modes, sizes) without touching disk.
- `--force` — overwrite a managed file that has been edited locally.
- `--all` — apply to every registered target (cannot be combined with positional names).

Example status output:

```
Agent installation status

[+] Claude Code
   - .mcp.json detected
[-] Cursor
   - Missing: .cursor/mcp.json
   - Hint: k6 x agent init cursor
...

[+] k6 MCP support
   - Found at /usr/local/bin/k6
```

## How it works

- **Skills are written once, in portable `SKILL.md` format.** Targets that consume `SKILL.md` natively (Claude Code, Codex CLI, Copilot, OpenCode) get them verbatim. Targets that don't (Cursor, Cline) get a thin wrapper around the same body.
- **MCP wiring is described once** in `agents/mcp/servers.yaml`. Each target adapter knows where its config file lives and what shape it expects.
- **Adapters compute a `Plan`; a shared `safefs` executes it.** That keeps adapters tiny and centralises the safety rules.

For the full design, see [`docs/DESIGN.md`](docs/DESIGN.md).

## Safety

- Never creates or modifies user-owned top-level files (`AGENTS.md`, `README.md`, etc.).
- Surgically merges shared JSON config (`.vscode/mcp.json`, `.cursor/mcp.json`, `opencode.json`, `.mcp.json`) — only the `k6` entry is touched; other servers and unrelated keys are preserved.
- Files inside `xk6-agent`-owned folders are stamped with an ownership marker. Re-running `init` is idempotent; `--force` is required to overwrite a file you have edited locally.
- `--dry-run` prints the full plan before any disk write.

Details: [`docs/DESIGN.md` §9](docs/DESIGN.md).

## Development

Requires Go ≥ 1.25.5.

```bash
git clone https://github.com/grafana/xk6-agent.git
cd xk6-agent
make test         # go test -race ./...
make lint         # golangci-lint (>= 1.60)
make vet
```

To try local changes against a real `k6 x agent` invocation, build a custom k6 binary with `xk6 build --with github.com/grafana/xk6-agent=.` and run it from the resulting `./k6`.

Repo layout:

- `agents/skills/` — `SKILL.md` source of truth (embedded via `//go:embed`).
- `agents/mcp/servers.yaml` — canonical MCP server schema.
- `agents/adapters/<target>/` — per-target adapters; each self-registers via `init()`.
- `agents/core/` — shared `Skill`, `MCPConfig`, and `safefs` helpers.
- `register.go`, `commands.go`, `status.go` — CLI surface.

To add a new target, create a package under `agents/adapters/<target>/` that implements `adapters.Target` and self-registers in `init()`. No edits to `register.go` are needed — the registry picks it up automatically.

## Troubleshooting

- **`folder already exists` / file collision** — re-run with `--force` to overwrite managed files. xk6-agent never deletes files it does not own.
- **`k6` not on PATH** (status reports `[-] k6 MCP support`) — install k6 from <https://grafana.com/docs/k6/latest/set-up/install-k6/>, and make sure your editor launches in a shell that can find it.
- **Cline MCP entry missing** — Cline's MCP config is global. Copy the snippet printed by `init cline` into `cline_mcp_settings.json`.

## Contributing

Issues and PRs are welcome — new target adapters and additional skills especially. Please run `make test` and `make lint` before submitting.

## License

Apache 2.0. See [`LICENSE`](LICENSE).
