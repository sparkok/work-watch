# work-watch Quickstart

> Requires [PilotDeck](https://github.com/OpenBMB/PilotDeck) — make sure it's running first.
>
**One command to batch-drive PilotDeck AI through your workflows.**

Once a power user sets up PilotDeck, there's only one thing you need to do: **write task files, then run.**

## Three Steps

### 1. Write Task Files

Create `*.txt` files under `tasks/<taskName>/jobs/` — one file per task you want the AI to handle:

```
tasks/my-task/jobs/
├── 001_requirements.txt
├── 002_design.txt
└── 003_implementation.txt
```

Use numeric prefixes to control execution order. File content is plain text — your instruction to the AI.

### 2. Run

```
work-watch my-task
```

The AI processes your txt files in order, with real-time progress display.

### 3. Collect Results

Export session records after completion:

```
work-watch export my-task        # JSON format
work-watch export my-task report # Report format
```

## Why Use It

- **You just write txt files** — put your tasks in files, work-watch handles the rest
- **Zero config for you** — PilotDeck is set up once by a power user, ready to go
- **Progress at a glance** — real-time status for every task
- **Resumable** — pick up where you left off, no duplicated work
- **Menu-driven** — forget commands? Just double-click `work-watch.exe`

---

**Turn what you don't want to do manually into txt files, and let AI handle it.**

