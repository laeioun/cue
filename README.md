# cue

Minimal Tab autocomplete for `git`, `gh`, and any CLI with parseable `--help` output.

```text
$ git com<Tab>
  commit    Record changes to the repository
```

`cue` installs a shell Tab hook. When you press Tab, the hook calls `cue complete "<buffer>" <cursor>`, shows a Bubble Tea picker on the terminal, and writes the selected completion back into your shell buffer.

## Install

```bash
go install github.com/laeioun/cue@latest
```

Then source the integration for your shell:

```bash
# bash
mkdir -p ~/.config/cue
curl -fsSL https://raw.githubusercontent.com/laeioun/cue/main/scripts/init.bash -o ~/.config/cue/init.bash
source ~/.config/cue/init.bash

# zsh
mkdir -p ~/.config/cue
curl -fsSL https://raw.githubusercontent.com/laeioun/cue/main/scripts/init.zsh -o ~/.config/cue/init.zsh
source ~/.config/cue/init.zsh
```

PowerShell:

```powershell
iwr https://raw.githubusercontent.com/laeioun/cue/main/scripts/init.ps1 -OutFile "$HOME\Documents\cue-init.ps1"
. "$HOME\Documents\cue-init.ps1"
```

For local development:

```bash
go build -o cue ./cmd/cue
mkdir -p ~/.local/bin
cp cue ~/.local/bin/
source scripts/init.bash
```

## Supported Shells

- bash
- zsh
- PowerShell with PSReadLine

## Supported Commands

- `git` from the bundled YAML spec
- `gh` from the bundled YAML spec
- Other commands through a best-effort `--help`, `-h`, then `help` fallback parser

Fallback completions are cached under your user cache directory, usually `~/.cache/cue`.

## Adding A Spec

Specs live in `specs/` and use recursive YAML nodes:

```yaml
name: example
description: Example CLI
subcommands:
  - name: run
    description: Run a task
    flags: [--verbose, -v]
```

Each node supports:

- `name`: completion text
- `description`: text shown in the picker
- `flags`: string list for MVP flag completions
- `subcommands`: nested command nodes

## Known Issue

The binary name `cue` conflicts with the CUE language CLI from `cuelang.org`. If both are installed, whichever appears first in `PATH` wins.
