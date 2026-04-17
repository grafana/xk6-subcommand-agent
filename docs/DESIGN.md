# xk6-agent — Design Document

Status: proposal · Audience: implementers (Claude Code, contributors) · Last updated: 2026-04-17

This document describes the next iteration of `xk6-agent`. It is meant to be read once,
then implemented. Decisions that have already been made are stated as facts; things that
still need confirmation are flagged as **Open Questions** at the end.

---

## 1. Purpose

`xk6-agent` is a `k6` extension that bootstraps an AI-assisted k6 testing workflow inside
a user's project. After running `k6 x agent init`, the user's preferred AI coding tool
should be able to:

1. Plan k6 test suites from natural-language requirements.
2. Generate idiomatic k6 scripts (smoke, load, stress, soak, spike, browser).
3. Validate and run those scripts via the `k6 x mcp` MCP server.
4. Convert existing Playwright tests to k6 browser tests.

The goal is to deliver this experience consistently across the major AI coding tools
without per-tool work each time the ecosystem shifts.

## 2. Goals & non-goals

### Goals

- **Author skills once, ship everywhere.** Skills are written as portable `SKILL.md`
  folders. Targets that natively understand the format consume them verbatim.
- **Adapter-per-target, registry-driven.** Adding a new target is one new package and
  one `init()` registration — no edits to a central switch.
- **Single canonical MCP schema.** The k6 MCP server is described once; per-target
  emitters generate the right config file in the right place.
- **Safe by default.** Never silently overwrite user-authored files. Files we manage
  are clearly marked. Re-running `init` is idempotent.
- **Discoverable without magic.** Skill descriptions drive auto-activation. We do not
  generate or modify any top-level project files (`AGENTS.md`, `README.md`, etc.) —
  see §3.

### Non-goals

- We are not building a skill *runtime*. We bootstrap files; the host agent (Claude
  Code, Codex CLI, Cursor, etc.) actually runs the skills.
- We are not maintaining one-off subagent definitions. Skills are the primary unit.
  On Claude Code we ship skills only — no `.claude/agents/` files.
- We are not converting prose between formats. If a target consumes `SKILL.md`
  natively, we copy it. If it doesn't, we copy the body and add the minimum native
  wrapper (no rewriting).

## 3. Background — why this design

The agent-tooling ecosystem converged in 2025–2026:

- **`SKILL.md` is now an open, cross-agent standard** consumed natively by Claude Code,
  OpenAI Codex CLI, GitHub Copilot (VS Code), Google Gemini CLI, JetBrains Junie,
  OpenCode, and 25+ other agents.
- **MCP** is the one truly portable wiring layer. Every serious agent supports MCP
  servers; only the *config file location and shape* differ per host.

This design treats `SKILL.md` as the source of truth and uses tiny adapters to handle
each target's idiosyncrasies (mostly: where do files go, and what does the MCP config
file look like).

### Why we do *not* touch `AGENTS.md`

`AGENTS.md` is now a widely-adopted Linux-Foundation-stewarded standard for top-level
project context. It would have been a natural place to advertise our skills and MCP
server. We deliberately do **not** create or modify it for two reasons:

1. **It is author-owned.** Users curate AGENTS.md as a hand-written description of
   their project. Tools that inject content into it — even with markers — risk
   conflicts, surprise diffs in PRs, and an awkward discoverability story
   ("why did this section appear?"). We avoid the whole class of issues by not
   touching it.
2. **It is not needed for any of our supported targets.** Every target in §4 either
   consumes `SKILL.md` natively (Claude Code, Codex CLI, Copilot, OpenCode) or has
   its own rules format that we already write to (Cursor, Cline). Skill activation
   happens via the `description` field, not via top-level prose.

Users who want a k6 section in their `AGENTS.md` can write one themselves; the
skill descriptions provide everything they'd need to mention. If a future target is
added that **only** reads `AGENTS.md` and has no skills/rules equivalent, this
decision should be revisited — but not before.

## 4. Supported targets (initial scope)

| Target | Skills location | Native `SKILL.md`? | MCP config | Entry-point notes |
|---|---|---|---|---|
| Claude Code | `.claude/skills/<name>/SKILL.md` | yes | `.claude/settings.local.json` (`enabledMcpJsonServers`) + `.mcp.json` | Skills only. No `.claude/agents/`. |
| OpenAI Codex CLI | `.codex/skills/<name>/SKILL.md` | yes | Codex config (per Codex docs) | Skills only; we do not touch `AGENTS.md`. |
| GitHub Copilot (VS Code) | `<copilot skills location>/<name>/SKILL.md` | yes | `.vscode/mcp.json` | See Open Question #2 for exact path. |
| OpenCode | `.opencode/skills/<name>/SKILL.md` | yes | `opencode.json` (`mcp.<name>`) | Existing target; refactor to use new layout. |
| Cursor | `.cursor/rules/<name>/` (frontmatter wrapper) | partial | `.cursor/mcp.json` | Body copied verbatim; YAML frontmatter wrapper added. |
| Cline | `.clinerules/<name>.md` (body + short header) | no | `cline_mcp_settings.json` (currently global) | No native skills; rules dir is the closest equivalent. |

A target's adapter is responsible for everything in its row.

## 5. Repository layout

```
xk6-agent/
├── agents/
│   ├── skills/                          # Source of truth — portable SKILL.md folders
│   │   ├── k6-test-planner/
│   │   │   ├── SKILL.md
│   │   │   └── (optional) scripts/, references/
│   │   ├── k6-load-test/SKILL.md
│   │   ├── k6-browser-test/SKILL.md
│   │   ├── k6-smoke-test/SKILL.md
│   │   └── k6-playwright-converter/SKILL.md
│   │
│   ├── mcp/
│   │   └── servers.yaml                 # Canonical MCP server schema
│   │
│   ├── adapters/                        # One subpackage per target family
│   │   ├── adapter.go                   # Target interface + registry
│   │   ├── registry.go                  # Self-registration via init()
│   │   ├── claude_code/
│   │   ├── codex_cli/
│   │   ├── vscode_copilot/
│   │   ├── opencode/
│   │   ├── cursor/
│   │   └── cline/
│   │
│   ├── templates/                       # Per-target wiring templates ONLY
│   │   ├── mcp/
│   │   │   ├── claude_settings_local.tmpl.json
│   │   │   ├── claude_mcp.tmpl.json
│   │   │   ├── vscode_mcp.tmpl.json
│   │   │   ├── cursor_mcp.tmpl.json
│   │   │   ├── opencode.tmpl.json
│   │   │   └── cline_settings.tmpl.json
│   │   └── frontmatter/
│   │       └── cursor_rule.tmpl.mdc
│   │
│   ├── core/                            # Shared types & helpers
│   │   ├── skill.go                     # Skill struct, parser, loader
│   │   ├── mcp.go                       # MCPConfig struct
│   │   └── safefs.go                    # Safe file ops (merge, ownership, idempotent)
│   │
│   ├── filesystem.go                    # Existing FS abstraction (kept)
│   ├── main.go                          # InitializeAgents() entry (refactored)
│   └── status.go                        # Existing status reporter (kept)
│
├── docs/
│   └── DESIGN.md                        # this file
│
├── register.go                          # k6 command registration (refactored)
└── ...
```

## 6. Source of truth: the `Skill`

A skill is a folder containing at minimum a `SKILL.md` file with YAML frontmatter
(`name`, `description`) and a markdown body. Optional sibling files (`scripts/`,
`references/`, `assets/`) are copied as-is.

```go
// agents/core/skill.go
type Skill struct {
    Name        string            // from frontmatter (kebab-case)
    Description string            // from frontmatter (used by host for activation)
    Body        string            // SKILL.md body (markdown)
    Frontmatter map[string]any    // raw frontmatter, preserved
    Files       []SkillFile       // sibling files: scripts/, references/, etc.
    Source      string            // embedded path, for diagnostics
    Overrides   *SkillOverrides   // optional, from sibling overrides.yaml
}

type SkillFile struct {
    RelPath string // path relative to the skill folder
    Content []byte
}

// Per-target hints, kept OUT of SKILL.md frontmatter to preserve portability.
// Adapters consult these only if relevant; absent ones are ignored.
type SkillOverrides struct {
    Cursor   *CursorOverride   `yaml:"cursor,omitempty"`   // globs, alwaysApply
    Copilot  *CopilotOverride  `yaml:"copilot,omitempty"`  // applyTo globs
    Cline    *ClineOverride    `yaml:"cline,omitempty"`
}
```

Skills are embedded into the binary via `//go:embed agents/skills/**`. The loader
walks the embedded FS, parses each `SKILL.md`, and returns a `[]Skill`.

**Authoring rule:** anything in `SKILL.md` must be portable. Target-specific knobs go
in `overrides.yaml` next to the `SKILL.md`, never in its frontmatter.

## 7. Canonical MCP schema

`agents/mcp/servers.yaml` is the single source of truth for which MCP servers we wire:

```yaml
# agents/mcp/servers.yaml
servers:
  k6:
    command: k6
    args: [x, mcp]
    description: k6 performance testing tools (validate, run, inspect scripts)
    env: {}
    transport: stdio
```

Loaded into:

```go
// agents/core/mcp.go
type MCPConfig struct {
    Servers map[string]MCPServer
}

type MCPServer struct {
    Command     string
    Args        []string
    Env         map[string]string
    Description string
    Transport   string // "stdio" | "http" | "sse"
}
```

Each adapter's `EmitMCP(MCPConfig) ([]File, error)` translates this into the right
file shape. Templates live under `agents/templates/mcp/`.

**Important:** when an MCP config file already exists in the user's project (e.g., the
user already has other MCP servers in `.vscode/mcp.json`), we do **not** overwrite the
file. We parse it, add/update only the entry for `k6`, leave everything else
untouched, and write back with original formatting preserved as much as practical
(see §9).

## 8. The `Target` interface and registry

```go
// agents/adapters/adapter.go
type Target interface {
    // Name is the canonical CLI name, e.g. "claude-code".
    Name() string

    // DisplayName is human-readable, e.g. "Claude Code".
    DisplayName() string

    // Capabilities reports what this target supports natively.
    Capabilities() Capabilities

    // Plan returns the set of files this target wants to write/modify
    // for the given inputs. It does NOT touch the filesystem.
    Plan(ctx Context, in Inputs) (Plan, error)
}

type Capabilities struct {
    NativeSkills    bool   // consumes SKILL.md verbatim
    SubagentDir     string // empty if not supported (we don't emit anyway)
    CommandsDir     string // for slash/prompt commands; empty if not supported
    MCPConfigPath   string // relative path of the MCP config file
}

type Inputs struct {
    Skills []core.Skill
    MCP    core.MCPConfig
    Root   string // user's project root
}

type Plan struct {
    Files   []PlannedFile
    Notices []string   // human-readable notes for the CLI
}

type PlannedFile struct {
    Path        string             // relative to Inputs.Root
    Content     []byte
    Mode        WriteMode          // CreateOnly | OverwriteIfManaged | MergeJSONByKey
    MergeKey    string             // for MergeJSONByKey: e.g. "mcpServers.k6"
    OwnerMarker string             // e.g. "xk6-agent:v1"
}

type WriteMode int
const (
    CreateOnly         WriteMode = iota // fail if exists & not ours
    OverwriteIfManaged                  // overwrite only if we own it (marker present)
    MergeJSONByKey                      // surgical JSON merge at MergeKey
)
```

### Registry

```go
// agents/adapters/registry.go
var registry = map[string]Target{}

func Register(t Target) { registry[t.Name()] = t }

func Get(name string) (Target, bool) { t, ok := registry[name]; return t, ok }

func All() []Target {
    out := make([]Target, 0, len(registry))
    for _, t := range registry { out = append(out, t) }
    sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
    return out
}
```

Each target package self-registers:

```go
// agents/adapters/claude_code/claude_code.go
func init() { adapters.Register(&claudeCode{}) }
```

This kills the hardcoded `supportedPlatforms` slice in `register.go`.

## 9. File safety — the `safefs` contract

This is the most important section. The tool must never silently overwrite a
user-authored file. Three categories of files exist:

1. **Files in user-owned, top-level locations** (`AGENTS.md`, `README.md`, etc.) —
   we never create or modify these. Full stop. See §3 for the rationale on
   `AGENTS.md` specifically.
2. **Shared structured config files** (`.vscode/mcp.json`, `.cursor/mcp.json`,
   `opencode.json`, `.mcp.json`, `.claude/settings.local.json`) — these may
   contain user-authored entries (other MCP servers, other settings). We
   surgically update only our own keys and never overwrite the file wholesale.
3. **Files inside our own folders** (`.claude/skills/k6-*/`, `.codex/skills/k6-*/`,
   `.cursor/rules/k6-*/`, `.clinerules/k6-*.md`) — these are owned by xk6-agent;
   we manage them with ownership markers and detect user edits.

Each category has its own write mode. Adapters never call the filesystem directly;
they declare intent through `PlannedFile` entries and `safefs.Apply(plan)` enforces
the rules.

### JSON / YAML / TOML config files (MCP, settings)

We own one logical key inside a possibly-shared file. Strategy:

1. Read the file (if it exists).
2. Parse with a format-preserving parser where practical (`encoding/json` won't
   preserve order/comments; consider `tidwall/sjson` for surgical edits to JSON,
   `goccy/go-yaml` with line preservation for YAML).
3. Set / replace only the specific path (`mcpServers.k6`, `mcp.k6`, etc.).
4. Write back. Comments and other entries preserved.

If parsing fails (file is malformed): **do not write**. Print a clear error and
suggest `--mcp=print` to dump the snippet for the user to merge manually.

`MergeJSONByKey` mode in `PlannedFile` exists for exactly this case. The adapter
declares the key (e.g., `"mcpServers.k6"`) and the snippet; `safefs` does the merge.

### Skill folders we generate (`.claude/skills/k6-*/`, etc.)

These folders are **owned by xk6-agent**. Policy:

1. **First write:** create the folder, drop a top-level marker file
   `.xk6-agent-managed` containing the version + the SHA of the source skill.
2. **Re-run, marker present, source SHA differs:** overwrite (skill content updated).
3. **Re-run, marker present, user-modified files inside:** detected via SHA mismatch
   on individual files. Print a warning per modified file; require `--force` to
   overwrite, otherwise skip with a notice.
4. **Marker absent (folder exists but isn't ours):** never overwrite. Error out and
   ask the user to remove the folder or pass `--target-dir` to relocate.

### Dry-run mode

`k6 x agent init <target> --dry-run` prints the full plan (file paths, modes, sizes,
which writes are merges vs. creates) without touching disk. **Required** before any
real run; the implementation must compute the `Plan` first, then either print or
execute.

## 10. CLI surface

```
k6 x agent init <target> [<target>...]   # initialize one or more targets
k6 x agent init --all                     # initialize every registered target
k6 x agent list                           # list available targets and skills
k6 x agent doctor [<target>]              # verify generated files match current schemas
k6 x agent update [<target>]              # re-run init, only updating managed regions
k6 x agent skills list                    # list skills shipped in the binary
k6 x agent skills show <name>             # print a skill's SKILL.md to stdout

Common flags:
  --dry-run             Print plan, do not write.
  --force               Overwrite user-modified managed files (skill folders only).
  --mcp=MODE            auto | merge | print | skip    (default: auto)
  --root <dir>          Project root (default: cwd).
  --skills-from <ref>   Use external skills source (file path, git ref). See §13.
```

`init` is the only command that writes files. `update` is `init` with
`OverwriteIfManaged` + `MergeJSONByKey` modes only — never creates new things you
didn't already accept.

## 11. End-to-end flow for `init`

```
1. Resolve targets from CLI args (validate against registry).
2. Load skills from embedded FS (or external --skills-from).
3. Load MCP schema from agents/mcp/servers.yaml.
4. For each target:
     a. Build Inputs{Skills, MCP, Root}.
     b. Call target.Plan(ctx, in) → Plan.
     c. If --dry-run: print Plan and continue.
     d. Else: hand Plan to safefs.Apply(plan).
        - safefs enforces all the safety rules in §9.
        - Returns per-file outcomes: created | updated | skipped | warned | errored.
5. Aggregate per-target outcomes; print summary table.
6. Exit 0 if no errors; non-zero if any target had errored writes.
```

The clean separation is: **adapters compute Plans, safefs executes them.**
This makes adapters trivial to test (no I/O) and centralises all safety logic.

## 12. Per-target adapter sketches

Each adapter is small. The shape:

```go
// agents/adapters/claude_code/claude_code.go
type claudeCode struct{}

func (claudeCode) Name() string        { return "claude-code" }
func (claudeCode) DisplayName() string { return "Claude Code" }

func (claudeCode) Capabilities() adapters.Capabilities {
    return adapters.Capabilities{
        NativeSkills:  true,
        MCPConfigPath: ".mcp.json",
    }
}

func (c claudeCode) Plan(ctx adapters.Context, in adapters.Inputs) (adapters.Plan, error) {
    var files []adapters.PlannedFile

    // 1. Drop each skill verbatim.
    for _, s := range in.Skills {
        files = append(files, planSkillFolder(".claude/skills", s)...)
    }

    // 2. MCP wiring.
    mcpFile, err := renderMCP("claude_mcp.tmpl.json", in.MCP)
    if err != nil { return adapters.Plan{}, err }
    files = append(files, adapters.PlannedFile{
        Path:        ".mcp.json",
        Content:     mcpFile,
        Mode:        adapters.MergeJSONByKey,
        MergeKey:    "mcpServers.k6",
        OwnerMarker: "xk6-agent:v1",
    })

    // 3. settings.local.json — enable our server.
    files = append(files, adapters.PlannedFile{
        Path:        ".claude/settings.local.json",
        Content:     mustRenderSettings(in.MCP),
        Mode:        adapters.MergeJSONByKey,
        MergeKey:    "enabledMcpJsonServers",
        OwnerMarker: "xk6-agent:v1",
    })

    return adapters.Plan{Files: files}, nil
}
```

Adapter responsibilities by target:

- **claude-code** — copy skills folders to `.claude/skills/`. Merge `mcpServers.k6` into
  `.mcp.json`. Merge `enabledMcpJsonServers` (add `"k6"`) into `.claude/settings.local.json`.
- **codex-cli** — copy skills folders to `.codex/skills/`. Merge MCP config (per Codex
  docs). Does **not** touch `AGENTS.md`; Codex reads native skills and that is enough
  for the planner to activate.
- **vscode-copilot** — copy skills folders to the Copilot skills location (Open
  Question #2). Merge `.vscode/mcp.json`. Optionally write `.github/instructions/k6.instructions.md`
  as a fallback for legacy Copilot setups.
- **opencode** — copy skills folders to `.opencode/skills/`. Merge `mcp.k6` into
  `opencode.json`.
- **cursor** — for each skill, render `.cursor/rules/<name>/RULE.md` (or `.mdc`) with
  YAML frontmatter (`description`, `globs` from overrides, `alwaysApply: false`) and
  the SKILL.md body verbatim. Merge `.cursor/mcp.json`.
- **cline** — for each skill, write `.clinerules/<name>.md` with a short header
  ("Loaded from k6-agent skill: …") then the SKILL.md body verbatim. MCP via
  `cline_mcp_settings.json` (currently global; print a notice).

`planSkillFolder` is a shared helper in `agents/adapters/internal/`:

```go
// Copies all files of a skill into <baseDir>/<skill.Name>/, including SKILL.md
// and any sibling scripts/references/. Adds .xk6-agent-managed marker file.
func planSkillFolder(baseDir string, s core.Skill) []adapters.PlannedFile { ... }
```

## 13. Loading skills from outside the binary (optional, stretch)

Embedded skills are the default. Power users may want to track skills out of band:

```
k6 x agent init claude-code --skills-from ./my-skills
k6 x agent init claude-code --skills-from github.com/grafana/k6-skills@v1.2.0
```

Implementation: the skill loader takes an `fs.FS`. Embedded source is one impl; a
filesystem source is another; a git-fetch source is a third (fetched into a cache
dir, then read as filesystem source). All three return `[]core.Skill`.

Defer this to a follow-up; design for it but don't implement until the in-binary
path is solid.

## 14. Migration plan (concrete, sequenced)

Each step is independently shippable.

**Step 1 — Land the new skill content.**
Create `agents/skills/` with five SKILL.md folders:

- `k6-test-planner/SKILL.md` (orchestrator, already provided)
- `k6-load-test/SKILL.md`
- `k6-browser-test/SKILL.md`
- `k6-smoke-test/SKILL.md`
- `k6-playwright-converter/SKILL.md` (port from existing `k6-playwright-test-converter.tmpl.md`,
  strip `{{.ConfigHeader}}` and any other Go-template syntax)

Add `agents/skills_embed.go` with `//go:embed skills/**`. No behaviour change yet.

**Step 2 — Introduce `core` package.**
Add `agents/core/skill.go` (Skill struct + parser/loader), `agents/core/mcp.go` (MCPConfig),
and `agents/mcp/servers.yaml` with the k6 server definition. Add a unit-tested loader.

**Step 3 — Introduce adapter interface + registry.**
Add `agents/adapters/adapter.go` and `registry.go`. Refactor the existing `claude/`,
`vscode/`, `opencode/` packages into `agents/adapters/{claude_code,vscode_copilot,opencode}/`,
implementing the new `Target` interface. Each new package self-registers via `init()`.

In `register.go`, replace the hardcoded `supportedPlatforms` and `defaultInitializerFactories`
with `adapters.All()` and `adapters.Get(name)`. Drop the old `Initializer` factory.

After this step, `init` should produce **identical files to today** for the three
existing targets — confirm with a snapshot test against the current `.claude/`,
`.github/`, `.opencode/` outputs in the repo.

**Step 4 — Introduce `safefs` + `Plan` execution.**
Build `agents/core/safefs.go`. Adapters now return a `Plan`; `safefs.Apply(plan)`
executes it with all the safety rules from §9. Implement `--dry-run`. This is the
highest-leverage piece: get it right and everything else gets safer for free.

**Step 5 — Add new targets.**
Implement `codex_cli`, `cursor`, `cline` adapters. Each is ~150–250 lines + tests.
For each, snapshot-test the generated output against a golden file.

**Step 6 — Add `doctor` and `update` commands.**
`doctor` re-plans without writing and reports drift. `update` re-runs `init` with
`OverwriteIfManaged`-only writes (no creates).

**Step 7 — CI + lint.**
Add a GitHub Actions workflow that runs `go test ./...`, `go vet`, and a
`go run ./cmd/agent-doctor` smoke test on a sample project per target.

**Step 8 (stretch) — External skill sources.**
Implement `--skills-from`. Document how teams can fork the k6 skills and consume
their own bundle.

## 15. Testing strategy

- **Unit tests** for the skill parser (frontmatter edge cases, missing fields,
  unicode names).
- **Unit tests** for `safefs` covering each `WriteMode`. The JSON-merge path is the
  most important branch — verify that other MCP servers, comments (where the parser
  preserves them), and unrelated keys are untouched.
- **Snapshot tests** per `(skill × target)` pair under `agents/adapters/<target>/testdata/`.
  Golden files are checked in. CI fails on diff.
- **Integration test** per target: run `init` against a temp directory, then run a
  second `init` and verify the operation is idempotent (no diffs).
- **No-touch test:** for every target, plant an `AGENTS.md` and a `README.md` with
  arbitrary content in the temp project root before running `init`. Assert their
  bytes are unchanged after `init`. This guards against regressions on the §3 promise.

Use `testscript` (`github.com/rogpeppe/go-internal/testscript`) for end-to-end CLI
tests — they're great for testing safety behaviours.

## 16. Backwards compatibility

- The existing two skills (`k6-test-generator`, `k6-playwright-test-converter`)
  are superseded by the new finer-grained skills. Keep the old `.tmpl.md` files
  in the repo for one release cycle, but don't ship them in `init` output. Note
  in the changelog that the planner + load/browser/smoke triplet replaces them.
- Existing users running `init` against their projects will see new skill folder
  names. The old folders (`.claude/agents/k6-test-generator.md`) will not be
  removed automatically — print a notice telling them they can `rm` the old
  files or pass `--remove-deprecated`.

## 17. Open questions (verify before coding)

1. **Cursor native SKILL.md:** ecosystem articles claim Cursor 2.2+ reads SKILL.md
   natively, but the official Cursor docs still emphasise `.cursor/rules`. Verify by
   testing on the latest Cursor; if native, the cursor adapter can simplify to a
   straight folder copy.
2. **VS Code / Copilot skills location:** "Use Agent Skills in VS Code" exists
   (https://code.visualstudio.com/docs/copilot/customization/agent-skills) but the
   exact path/naming should be confirmed against current docs before implementing
   the adapter.
3. **Codex CLI MCP config path/shape:** confirm against current Codex docs; the
   API has been moving.
4. **Cline project-scope MCP:** as of latest research, Cline's MCP config is global
   only (`cline_mcp_settings.json`). If project-scope ships, switch the adapter.
   Until then, the adapter prints a notice instructing the user to add the snippet
   manually, or `--mcp=print` outputs the snippet.
5. **`skillkit` / `skillport`:** these existing tools translate skills between
   agent formats. Worth a 1-day spike to evaluate whether to wrap one of them
   instead of building our own translation in the cursor/cline adapters.
6. **Plugin marketplace publication:** Claude Code now has a plugin marketplace.
   Decide whether to also publish the k6 skills as an installable plugin
   (`/plugin install grafana/k6-skills`) in a follow-up.

---

## Appendix A — Example MCP snippets per target

```jsonc
// .mcp.json (Claude Code) and .vscode/mcp.json (Copilot)
{
  "mcpServers": {
    "k6": {
      "command": "k6",
      "args": ["x", "mcp"],
      "type": "stdio"
    }
  }
}
```

```jsonc
// .cursor/mcp.json
{
  "mcpServers": {
    "k6": { "command": "k6", "args": ["x", "mcp"] }
  }
}
```

```jsonc
// opencode.json (partial)
{
  "mcp": {
    "k6": { "type": "local", "command": ["k6", "x", "mcp"] }
  }
}
```

## Appendix B — Implementation order checklist (for the implementer)

- [ ] Step 1: skills/ folders + `//go:embed`
- [ ] Step 2: core.Skill + core.MCPConfig + servers.yaml + loader tests
- [ ] Step 3: adapter interface + registry + refactor existing 3 adapters
- [ ] Step 3a: snapshot test confirming bit-identical output for existing targets
- [ ] Step 4: safefs + Plan + --dry-run + no-touch tests
- [ ] Step 5a: codex_cli adapter + tests
- [ ] Step 5b: cursor adapter + tests
- [ ] Step 5c: cline adapter + tests
- [ ] Step 6: doctor + update commands
- [ ] Step 7: CI workflow
- [ ] Step 8 (stretch): --skills-from external sources
