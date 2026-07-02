# Work-Watch Manual

> Requires [PilotDeck](https://github.com/OpenBMB/PilotDeck) — make sure it's running first.

Work-Watch is a task supervisor tool that works with PilotDeck. It submits pre-written task instructions to PilotDeck one by one, tracks progress automatically, and confirms results after completion.

## Quick Start

Double-click `work_watch.exe` to launch menu mode, or run from command line:

```
work_watch <taskName>
```

On first launch, PilotDeck configuration (`~/.pilotdeck/pilotdeck.yaml`) is auto-detected and `config.yaml` is generated — no manual setup required.

---

## Directory Structure

```
work-watch/
├── config.yaml              # Global config (PilotDeck connection)
├── work_watch.exe           # Executable
├── tasks/                   # Task directory
│   ├── taskA/               # One subdir per task
│   │   ├── task.yaml        # Task-level config (debug mode, session_id)
│   │   ├── status           # YAML status file (auto-managed)
│   │   ├── jobs/            # Task instruction files (*.txt)
│   │   │   ├── 001_xxx.txt
│   │   │   └── 002_yyy.txt
│   │   ├── logs/            # Execution logs
│   │   └── export/          # Export results
│   └── taskB/
└── doc/
    ├── quickstart.md
    ├── quickstart_cn.md
    ├── manual.md
    └── manual_cn.md
```

### Global Config `config.yaml`

```yaml
pilotdeck:
  base_url: http://localhost:3001    # PilotDeck server address
  api_key: <server-token>            # Auto-read from ~/.pilotdeck/server-token
  project_path: E:/study/ai/PilotDeck
```

> If `config.yaml` doesn't exist at startup, it's auto-generated from `~/.pilotdeck/pilotdeck.yaml` and `~/.pilotdeck/server-token`.

### Task Config `tasks/<taskName>/task.yaml`

```yaml
debug: true                      # Debug mode
session_id: "abc123"             # Last session ID (auto-managed)
```

---

## Task Files

Each task is a directory. Put `*.txt` files under `jobs/` — each file is one instruction sent to PilotDeck.

Files are executed in **sorted order** by filename (use numeric prefixes):

```
jobs/
├── 001_requirements.txt
├── 002_design.txt
└── 003_implementation.txt
```

File content is plain text, sent directly as a message to PilotDeck.

---

## Usage

### 1. Menu Mode (no arguments)

Double-click `work_watch.exe` or run `work_watch`:

```
========== Work-Watch Task Supervisor ==========
 1. Configure   — create or modify task config
 2. Execute     — run a task (async)
 3. Export      — export task session records
 4. Status      — view task status
 5. Reset       — revert task to initial state
 6. Exit
```

- **1 Configure**: modify existing task or create a new one
- **2 Execute**: run a task asynchronously without blocking the menu
- **3 Export**: export session records as JSON / Markdown
- **4 Status**: summary of all tasks
- **5 Reset**: clear all progress and start fresh

### 2. Command-line Mode

| Command | Description |
|---|---|
| `work_watch` | Interactive menu |
| `work_watch <taskName>` | Run specified task directly |
| `work_watch config <taskName>` | Configure specified task |
| `work_watch export <taskName>` | Export session as JSON |
| `work_watch export <taskName> report` | Export session as Markdown report |
| `work_watch export <taskName> detail` | Export full transcript as Markdown |
| `work_watch status` | View all task statuses |
| `work_watch reset <taskName>` | Reset task to initial state |

#### Examples

```bash
# Create and run a new task
work_watch my-task

# View all task statuses
work_watch status

# Export a task's session report
work_watch export my-task report
```

---

## Execution Flow

```
1. Detect config
   ├── task.yaml missing → launch config wizard
   │   └── auto-refresh PilotDeck connection after wizard
   ├── config.yaml missing → auto-init from ~/.pilotdeck/
   └── check PilotDeck server is online

2. Execute jobs sequentially
   ┌── read next incomplete *.txt file
   ├── submit to PilotDeck
   ├── log to logs/
   ├── mark as completed (write status file)
   └── repeat until all jobs done

3. Result confirmation
   ├── check if task was already confirmed
   ├── wait for PilotDeck session to finish processing (WebSocket)
   ├── fetch session messages to verify content exists
   └── mark confirmed in status file
```

### Resume from Interruption

If a task is interrupted (Ctrl+C, timeout), completed jobs are **not** re-executed. The next run picks up from the first incomplete job.

To restart from scratch, delete the `status` file or use `work_watch reset <taskName>`.

### Result Confirmation

After all jobs complete, the tool waits for the PilotDeck session to finish processing via WebSocket (`check-session-status`), then fetches the session messages to confirm:

- Messages exist → task succeeded
- No messages → task failed

Once confirmed, `confirmed: true` is written to the status file and the task won't be re-confirmed on subsequent runs.

### Duplicate Run Prevention

If you try to run a task that already has an active PilotDeck session (with incomplete jobs), you'll see:

```
任务正在执行中 (session: xxx)，请等待完成后再试
```

The run is blocked to prevent double submission.

---

## Status View

Run `work_watch status` or select "Status" from the menu:

```
--- Task Status ---
  my-first-task (未配置 / not configured)
  second-task [2/3 jobs, session: abc123, 执行中 / running]
  third-task [3/3 jobs, session: def456]
```

| Status | Meaning |
|---|---|
| `(未配置)` | `task.yaml` doesn't exist |
| `[2/3 jobs]` | 2 of 3 jobs completed |
| `[3/3 jobs, session: xxx]` | All complete, with session ID |
| `执行中` | Session active, jobs still running |
| `(配置错误: ...)` | Config file has an issue |

---

## Export Results

After a task completes, export the PilotDeck session records:

### JSON Export (raw data)

```bash
work_watch export my-task
```

Outputs to `tasks/my-task/export/session-<id>-<timestamp>.json`

### Report Export (summary)

```bash
work_watch export my-task report
```

Markdown summary including:
- User request list
- Assistant reply summaries
- Tool call records
- Error information

### Detail Export (full transcript)

```bash
work_watch export my-task detail
```

Markdown document with complete message content.

---

## Configuration Details

### Global Config (`config.yaml`)

Stores PilotDeck connection info. Populated in this priority order:

1. **Environment variables**: `HOST` + `PORT` → base_url, `API_KEY` → api_key, `PROJECT_PATH` → project_path
2. **Auto-discovery**: reads port from `~/.pilotdeck/pilotdeck.yaml` → `http://localhost:<port>`, API key from `~/.pilotdeck/server-token`
3. **Interactive wizard**: use "Configure" in the menu
4. **Default**: `http://localhost:3001`

### Environment Variables

| Variable | Description |
|---|---|
| `HOST` | PilotDeck server host |
| `PORT` | PilotDeck server port |
| `API_KEY` | API key |
| `PROJECT_PATH` | PilotDeck project path |
| `PILOT_HOME` | PilotDeck config directory (default `~/.pilotdeck`) |

---

## FAQ

### Q: "PilotDeck 服务未启动" at startup

Make sure PilotDeck is running. Check that `base_url` in `config.yaml` is correct (default `http://localhost:3001`).

### Q: "Invalid or inactive API key"

API key expired or mismatched. Deleting `task.yaml` and reconfiguring will auto-refresh from `~/.pilotdeck/server-token`. You can also manually update `api_key` in `config.yaml`.

### Q: How to re-run a task from scratch

Use `work_watch reset <taskName>` or delete `tasks/<taskName>/status`.

### Q: Task shows "未配置" (not configured)

`task.yaml` doesn't exist. Running the task will auto-launch the wizard, or use `work_watch config <taskName>`.

### Q: Task was interrupted mid-execution

Completed jobs are saved in the `status` file. Re-running continues from the breakpoint.
