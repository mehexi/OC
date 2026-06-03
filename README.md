# oc — Terminal AI coding assistant

oc is a TUI-based AI coding assistant with vim-like keybindings, session history, and interactive command menus.

## Install

### One-liner

```sh
curl -fsSL https://raw.githubusercontent.com/mehexi/OC/main/scripts/install.sh | bash
```

### Manual

Download the tarball for your platform from [releases](https://github.com/mehexi/OC/releases), extract, and place `oc` in your PATH.

### Build from source

```sh
go install github.com/mehexi/OC/cmd/oc@latest
```

Requires Go 1.26+.

## Usage

```
oc
```

Starts the local server and opens the TUI. Type your question and press Enter. The assistant responds inline.

## Keybindings

| Mode | Key | Action |
|------|-----|--------|
| — | `/` | Show command menu |
| — | `Esc` | Normal mode |
| Normal | `i` | Insert mode |
| Normal | `j`/`k` | Scroll down/up |
| Normal | `gg`/`G` | Scroll to top/bottom |
| Normal | `V` | Visual mode (select messages) |
| Visual | `y` | Yank selected messages |
| Insert | `Enter` | Send message |
| Insert | `Ctrl+C` | Clear input / Quit |

## Commands

| Command | Action |
|---------|--------|
| `/sessions` | Browse past sessions (j/k navigate, Enter load) |
| `/session new` | Start a fresh session |
| `/load <n>` | Load session by number |

## Sessions

Chat history is stored locally and automatically. Use `/sessions` to browse and reload previous conversations. Each session shows its title, token usage, and context limit.

## Development

```sh
git clone https://github.com/mehexi/OC.git
cd OC
go build ./cmd/oc
./oc
```
