# work-watch Quickstart

> Requires [PilotDeck](https://github.com/OpenBMB/PilotDeck) — make sure it's running first.
>
> Typical setup: a power user configures PilotDeck once; everyone else just runs work-watch.

**One command to batch-drive PilotDeck AI through your workflows.**

Tired of manually sending requests to AI, waiting for replies, then sending the next one? Write your tasks as a job list, let `work-watch` submit them one by one, and just collect the results.

## Three Steps

1. **Configure** — list the jobs to execute
2. **Run** — `work-watch <taskName>`
3. **Collect** — grab a coffee and check the session records

## Why Use It

- **Zero config** — auto-discovers PilotDeck connection info, works out of the box
- **Configure once, run repeatedly** — write task config once, reset session to re-run
- **Progress at a glance** — real-time job status and session ID for each job
- **Flexible export** — JSON for processing, Markdown reports for archiving, full transcripts for review
- **Menu-driven** — forget commands? Just run `work-watch` for the interactive menu
- **Resumable** — session state auto-saved; pick up where you left off

## Quick Start

```bash
go build -o work-watch.exe
work-watch                  # interactive menu
work-watch myTask           # run a task directly
work-watch export myTask    # export session records
```

---

**Turn your repetitive work into a job list and let AI handle it.**
