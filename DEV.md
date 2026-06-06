# oc — Terminal AI coding assistant

## Build & dev

- **Build**: `go build -o oc ./cmd/oc/`
- **Hot reload**: uses `air` (`.air.toml`), `air` rebuilds to `./tmp/main`
- **Go version**: 1.26.3+ (go.mod)
- **No tests** in repo — no `_test.go` files anywhere
- **No lint/typecheck scripts** — standard Go tooling only
- **Spelling**: `IntialModel` (not `InitialModel`) in `internal/tui/model.go:106` — intentional name, do not "fix"

## Architecture

- **Module**: `oc` (go.mod)
- **Entrypoint**: `cmd/oc/main.go` — creates Bubbletea v2 program with `tui.IntialModel()`
- **TUI framework**: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`, `charm.land/glamour/v2`
- **External server**: TUI is a client that spawns/manages `opencode serve` as a subprocess. Server URL: `http://127.0.0.1:4096`

### Package map

| Package | Responsibility |
|---------|---------------|
| `cmd/oc/` | Entrypoint |
| `internal/tui/` | Bubbletea Model, Update, View, keybinding modes |
| `internal/tui/commands/` | Cmds (side effects) and message types |
| `internal/api/` | HTTP client to opencode server |
| `internal/server/` | Manages `opencode serve` lifecycle (start/kill) |
| `internal/history/` | Session persistence to `~/.oc/sessions/` |
| `internal/domain/` | Shared types (`Message`, `Session`, `Config`) |
| `internal/sysprompt/` | Multi-agent judge system prompt |

### Vim modes (`internal/tui/model.go`)

`modeInsert`, `modeNormal`, `modeVisual`, `modeQus` (questions), `modeSession` (browse), `modeCmd`, `modePerm` (permissions). Dispatched via `onKeyPress` → typed handler per mode. Handlers are on `Model` value receivers returning `(Model, tea.Cmd)`.

### Message flow

All messages defined in `internal/tui/commands/messages.go`. Update dispatches via type switch in `internal/tui/update.go:377`. SSE listener (`internal/tui/commands/sse.go`) subscribes to `/global/event` and uses `program.Send()` for interactive messages (questions, permissions, streaming deltas).

### SSE event types consumed

- `question.asked` → `ControlRequestMsg`
- `permission.asked` → `PermissionRequestMsg`
- `message.part.delta` → `ChatStreamMsg` (text/reasoning chunks)
- `message.part.updated` → resolves buffered deltas by part type
- `message.updated` → `ChatStreamMsg{Done: true}` when finish reason present

### Key client API endpoints

- `GET /global/health` — health check
- `GET /config/providers` — provider list
- `GET /path` — working directory
- `POST /session` — create session
- `GET /session/:id` — session details (tokens, limits)
- `POST /session/:id/message` — send message
- `GET /global/event` — SSE event stream
- `POST /permission/:id/reply` — reply to permission request
- `POST /question/:id/reply` — reply to question

### Commands (`internal/tui/commands/`)

- Events bubble as messages defined in `messages.go`. Use `program.Send()` for messages that need to be dispatched while SSE listener blocks.

## Keybindings (from README)

| Mode | Key | Action |
|------|-----|--------|
| — | `/` | Show command menu |
| — | `Esc` | Normal mode |
| Normal | `i` | Insert mode |
| Normal | `j/k` | Scroll down/up |
| Normal | `gg`/`G` | Scroll to top/bottom |
| Normal | `V` | Visual mode (select messages) |
| Visual | `y` | Yank selected messages |
| Insert | `Enter` | Send message |
| Insert | `Ctrl+C` | Clear input / Quit |

### Commands

`/help`, `/sessions`, `/session new`, `/clear`, `/multiagent`, `/retry`, `/load <n>`, `/tokens`, `/exit`

## Release

- GitHub Actions: `.github/workflows/release.yml` — cross-compiles linux/darwin amd64/arm64, tarballs + checksums, triggered on `v*` tag
- Install script: `scripts/install.sh`
- Build output ignored: `/dist/`, `/tmp/`, `/os`, `/oc` (`.gitignore`)
