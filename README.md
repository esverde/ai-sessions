# ais

[English](README.md) | [简体中文](README.zh-CN.md)

`ais` is a small cross-platform TUI for finding and resuming local Claude Code, OpenAI Codex, and Antigravity sessions.

It is intentionally read-only: the program scans native session stores, shows a searchable list, and launches the selected native CLI with the session's recorded working directory.

## Features

- Cross-platform Go + Bubble Tea TUI.
- Scans Claude Code, OpenAI Codex, and Antigravity (`agy`) sessions.
- Lists sessions from the current directory or from all projects.
- Provider filter: `all`, `claude`, `codex`, or `antigravity`.
- Sort by recent activity or project path.
- JSON configuration file with no settings GUI.
- Search with `/`, provider switching with `p`, scope switching with `a`, and sort switching with `s`.
- Resume with `Enter`: launches `claude --resume <id>`, `codex resume <id>`, or `agy --conversation <id>` from the session's recorded working directory.
- Supports active Claude/Codex/Antigravity sessions and archived Codex sessions.

## Download, build, and install

Prebuilt binaries are available from the [GitHub Releases](https://github.com/esverde/ai-sessions/releases) page.

The installed command is `ais` on Windows, macOS, and Linux.

```powershell
go build ./cmd/ais
```

For regular use, install the command into Go's binary directory and make sure that directory is on `PATH`:

```powershell
go install ./cmd/ais
ais
```

### macOS and Linux release binaries

Downloaded macOS and Linux binaries may need the executable bit enabled. For example:

```sh
chmod +x ./ais-v0.1.0-darwin-arm64
mkdir -p ~/.local/bin
mv ./ais-v0.1.0-darwin-arm64 ~/.local/bin/ais
export PATH="$HOME/.local/bin:$PATH"
ais
```

Use the matching Linux or Intel macOS asset in place of `ais-v0.1.0-darwin-arm64`. You can add the `PATH` export to your shell profile for future sessions.

### Windows release binary

Place the downloaded Windows binary on `PATH` (you may rename it to `ais.exe`). Then run:

```powershell
ais
```

Run it from the directory you want to inspect. The default scope is the current directory and its descendants. Useful overrides:

```powershell
ais --all
ais --provider claude
ais --provider codex
ais --provider antigravity
ais --sort path
ais --init-config
```

The command name is `ais` (AI Sessions).

## TUI keys

| Key | Action |
| --- | --- |
| `Enter` | Resume the selected session |
| `/` | Search/filter sessions |
| `a` | Toggle current directory/all projects |
| `p` | Cycle all/Claude/Codex/Antigravity |
| `s` | Toggle active/path sorting |
| `r` | Refresh the scan |
| `?` | Show the key summary |
| `q` / `Ctrl+C` | Quit |

## Configuration

The configuration file is created with:

```powershell
ais --init-config
```

Default locations use the operating system's user config directory. Set `AIS_CONFIG` or pass `--config` to use a different file.

Example:

```json
{
  "provider": "all",
  "scope": "current",
  "sort": "active",
  "include_archived": false,
  "preview_length": 160,
  "max_sessions": 200
}
```

Accepted values:

- `provider`: `all`, `claude`, `codex`, `antigravity` (`agy` is accepted as an alias)
- `scope`: `current`, `all`
- `sort`: `active`, `path`

Provider roots can be overridden by the native environment variables `CLAUDE_CONFIG_DIR` and `CODEX_HOME`. Antigravity has no native equivalent, so `ais` provides `ANTIGRAVITY_HOME` for the same purpose.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).

## Session sources

Claude sessions are read from `.claude/projects/**/*.jsonl`. Codex sessions are read from `.codex/sessions/**/*.jsonl`; archived Codex sessions are included when `include_archived` is enabled.

Antigravity (formerly Gemini CLI) keeps each conversation in its own SQLite store under `.gemini/antigravity-cli/conversations/<id>.db`, and records which workspace an interactive conversation belongs to in `.gemini/antigravity-cli/history.jsonl`. `ais` reads that history file first, and falls back to scanning the conversation store for the workspace URI when a conversation is missing from it.

Only Antigravity CLI conversations are listed. The IDE keeps its own conversations under `.gemini/antigravity/`, but `agy --conversation <id>` refuses them with `trajectory not found`, so listing them would only offer sessions that cannot be resumed.

When you press `Enter`, `ais` passes the selected session ID to the native CLI and starts it in the recorded project directory. It does not edit or delete the native session files.

Unknown or malformed JSONL records are ignored. A conversation whose working directory cannot be determined is skipped rather than resumed in the wrong place.
