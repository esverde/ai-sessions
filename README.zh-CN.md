# ais

[English](README.md) | [简体中文](README.zh-CN.md)

`ais` 是一个跨平台 TUI，用于查找和恢复本机的 Claude Code 与 OpenAI Codex 会话。

它保持只读：程序扫描原生 JSONL 会话文件，展示可搜索的会话列表，并使用会话记录的工作目录启动对应的原生 CLI。

## 下载

预编译版本可以在 [GitHub Releases](https://github.com/esverde/ai-sessions/releases) 页面下载。

## 构建

```powershell
go build -o ais.exe ./cmd/ais
```

也可以将命令安装到 Go 的二进制目录：

```powershell
go install ./cmd/ais
```

在希望检查的目录中运行：

```powershell
./ais.exe
```

默认范围是当前目录及其子目录。常用覆盖参数：

```powershell
./ais.exe --all
./ais.exe --provider claude
./ais.exe --provider codex
./ais.exe --sort path
./ais.exe --init-config
```

命令名是 `ais`，代表 AI Sessions。

## TUI 快捷键

| 按键 | 操作 |
| --- | --- |
| `Enter` | 恢复选中的会话 |
| `/` | 搜索/筛选会话 |
| `a` | 切换当前目录/全部项目 |
| `p` | 循环切换全部/Claude/Codex |
| `s` | 切换按活跃时间/路径排序 |
| `r` | 重新扫描 |
| `?` | 显示快捷键摘要 |
| `q` / `Ctrl+C` | 退出 |

## 配置

使用以下命令创建配置文件：

```powershell
./ais.exe --init-config
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

- `provider`：`all`、`claude`、`codex`
- `scope`：`current`、`all`
- `sort`：`active`、`path`

会话根目录可以通过原生环境变量 `CLAUDE_CONFIG_DIR` 和 `CODEX_HOME` 覆盖。

## 许可证

本项目采用 MIT 许可证，详见 [LICENSE](LICENSE)。

## 会话来源

Claude 会话读取自 `.claude/projects/**/*.jsonl`。Codex 会话读取自 `.codex/sessions/**/*.jsonl`；启用 `include_archived` 后也会包含已归档的 Codex 会话。

未知或格式错误的 JSONL 记录会被忽略。程序不会修改或删除原生会话文件。
