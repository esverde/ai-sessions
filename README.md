# ais

`ais` is a small cross-platform TUI for finding and resuming local Claude Code and OpenAI Codex sessions.

It is intentionally read-only: the program scans native JSONL files, shows a searchable list, and launches the selected native CLI with the session's recorded working directory.

## Build

```powershell
go build -o ais.exe ./cmd/ais
```

Or install the command into Go's binary directory:

```powershell
go install ./cmd/ais
```

Run it from the directory you want to inspect:

```powershell
./ais.exe
```

The default scope is the current directory and its descendants. Useful overrides:

```powershell
./ais.exe --all
./ais.exe --provider claude
./ais.exe --provider codex
./ais.exe --sort path
./ais.exe --init-config
```

The command name is `ais` (AI Sessions).

## TUI keys

| Key | Action |
| --- | --- |
| `Enter` | Resume the selected session |
| `/` | Search/filter sessions |
| `a` | Toggle current directory/all projects |
| `p` | Cycle all/Claude/Codex |
| `s` | Toggle active/path sorting |
| `r` | Refresh the scan |
| `?` | Show the key summary |
| `q` / `Ctrl+C` | Quit |

## Configuration

The configuration file is created with:

```powershell
./ais.exe --init-config
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

- `provider`: `all`, `claude`, `codex`
- `scope`: `current`, `all`
- `sort`: `active`, `path`

Provider roots can be overridden by the native environment variables `CLAUDE_CONFIG_DIR` and `CODEX_HOME`.

## Session sources

Claude sessions are read from `.claude/projects/**/*.jsonl`. Codex sessions are read from `.codex/sessions/**/*.jsonl`; archived Codex sessions are included when `include_archived` is enabled.

Unknown or malformed JSONL records are ignored. The program never edits or deletes native session files.
