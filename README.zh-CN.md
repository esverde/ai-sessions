# ais

[English](README.md) | [简体中文](README.zh-CN.md)

`ais` 是一个跨平台 TUI，用于查找和恢复本机的 Claude Code、OpenAI Codex 与 Antigravity 会话。

它保持只读：程序扫描各家原生的会话存储，展示可搜索的会话列表，并使用会话记录的工作目录启动对应的原生 CLI。

## 功能

- 跨平台 Go + Bubble Tea TUI。
- 扫描 Claude Code、OpenAI Codex 与 Antigravity（`agy`）会话。
- 支持列出当前目录及其子目录，或全部项目中的会话。
- provider 筛选：`all`、`claude`、`codex`、`antigravity`。
- 支持按最近活跃时间或项目文件夹路径排序。
- 使用 JSON 配置文件，不提供设置 GUI。
- `/` 搜索，`p` 切换 provider，`a` 切换目录范围，`s` 切换排序方式。
- 按 `Enter` 恢复会话：在会话记录的工作目录中调用 `claude --resume <id>`、`codex resume <id>` 或 `agy --conversation <id>`。
- 支持 Claude/Codex/Antigravity 活跃会话，以及 Codex archived 会话。

## 下载

预编译版本可以在 [GitHub Releases](https://github.com/esverde/ai-sessions/releases) 页面下载。

## 下载、构建和安装

在 Windows、macOS 和 Linux 上，安装后的命令统一使用 `ais`。

```sh
go build ./cmd/ais
```

日常使用时，建议将命令安装到 Go 的二进制目录，并确保该目录已经加入 `PATH`：

```sh
go install ./cmd/ais
ais
```

### macOS 和 Linux 预编译版本

下载的 macOS/Linux 二进制文件可能需要增加可执行权限。例如：

```sh
chmod +x ./ais-v0.1.0-darwin-arm64
mkdir -p ~/.local/bin
mv ./ais-v0.1.0-darwin-arm64 ~/.local/bin/ais
export PATH="$HOME/.local/bin:$PATH"
ais
```

请将示例中的 `ais-v0.1.0-darwin-arm64` 替换为对应的 Linux 或 Intel Mac 版本。可以把 `PATH` 设置加入 shell 配置文件，避免每次重新设置。

### Windows 预编译版本

将下载的 Windows 二进制文件放到 `PATH` 中，也可以将文件重命名为 `ais.exe`。之后统一运行：

```powershell
ais
```

在希望检查的目录中运行。默认范围是当前目录及其子目录。常用覆盖参数：

```powershell
ais --all
ais --provider claude
ais --provider codex
ais --provider antigravity
ais --sort path
ais --init-config
```

命令名是 `ais`，代表 AI Sessions。

## TUI 快捷键

| 按键 | 操作 |
| --- | --- |
| `Enter` | 恢复选中的会话 |
| `/` | 搜索/筛选会话 |
| `a` | 切换当前目录/全部项目 |
| `p` | 循环切换全部/Claude/Codex/Antigravity |
| `s` | 切换按活跃时间/路径排序 |
| `r` | 重新扫描 |
| `?` | 显示快捷键摘要 |
| `q` / `Ctrl+C` | 退出 |

## 配置

使用以下命令创建配置文件：

```powershell
ais --init-config
```

默认位置使用操作系统的用户配置目录。也可以设置 `AIS_CONFIG` 或传入 `--config` 使用其他文件。

示例：

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

支持的值：

- `provider`：`all`、`claude`、`codex`、`antigravity`（也接受别名 `agy`）
- `scope`：`current`、`all`
- `sort`：`active`、`path`

会话根目录可以通过原生环境变量 `CLAUDE_CONFIG_DIR` 和 `CODEX_HOME` 覆盖。Antigravity 没有对应的原生变量，`ais` 为它提供了 `ANTIGRAVITY_HOME`。

## 许可证

本项目采用 MIT 许可证，详见 [LICENSE](LICENSE)。

## 会话来源

Claude 会话读取自 `.claude/projects/**/*.jsonl`。Codex 会话读取自 `.codex/sessions/**/*.jsonl`；启用 `include_archived` 后也会包含已归档的 Codex 会话。

Antigravity（原 Gemini CLI）把每个会话单独存成一个 SQLite 库，位于 `.gemini/antigravity-cli/conversations/<id>.db`，而"某个交互式会话属于哪个工作目录"记录在 `.gemini/antigravity-cli/history.jsonl`。`ais` 优先读这份历史文件；会话不在其中时，再从会话库的字节里回退查找工作目录 URI。

只列出 Antigravity CLI 的会话。IDE 的会话存在 `.gemini/antigravity/` 下，但 `agy --conversation <id>` 会以 `trajectory not found` 拒绝它们——列出来只会给出恢复不了的条目。

按下 `Enter` 后，`ais` 会把选中的会话 ID 传给对应的原生 CLI，并在会话记录的项目目录中启动。程序不会修改或删除原生会话文件。

未知或格式错误的 JSONL 记录会被忽略。无法确定工作目录的会话会被跳过，而不是在错误的位置恢复。
