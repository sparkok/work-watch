# work-watch

> Requires [PilotDeck](https://github.com/OpenBMB/PilotDeck) — make sure it's running first.
>
> Typical setup: a power user configures PilotDeck once; everyone else just runs work-watch.

A command-line tool for batch-driving PilotDeck AI Agent through multi-job workflows.

## Features

- **Batch execution** — runs multiple tasks, submitting each job to PilotDeck AI in sequence
- **Interactive menu** — configure, run, export, check status, and reset tasks without memorizing commands
- **Progress tracking** — real-time job status and session ID for each submission
- **Session export** — JSON / Markdown report / full transcript in three export formats
- **Auto-discovery** — reads PilotDeck connection info on first run, zero manual config

## Usage

```
work-watch                   # interactive menu
work-watch <taskName>        # run a task directly
work-watch config <taskName> # configure a task
work-watch status            # view all task statuses
work-watch export <taskName> # export session (default JSON)
work-watch reset <taskName>  # reset task session
```

## Configuration

On first run, PilotDeck connection info is auto-read from `~/.pilotdeck/` to generate `config.yaml`. Each task directory has its own `task.yaml` for session state.
