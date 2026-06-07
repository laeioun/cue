# cue

Minimal Tab autocomplete that hooks into your shell's normal Tab key. No session wrapper, no `is` prefix, no new shell mode.

```text
$ git com<Tab>
  commit    Record changes to the repository
```

`cue` installs a shell Tab hook. When you press Tab, the hook calls `cue complete "<buffer>" <cursor>`, expands aliases, fast-accepts clear matches, shows a Bubble Tea picker when needed, and writes the selected completion back into your shell buffer.

## Install

Install the latest pre-built binary and shell integration:

```bash
curl -fsSL https://raw.githubusercontent.com/laeioun/cue/main/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/laeioun/cue/main/install.ps1 | iex
```

The installer downloads the latest GitHub Release for your platform, places `cue`
on your user PATH, and adds the shell hook.

### Manual Shell Setup

If you already have the binary installed, run:

```bash
cue install
```

On PowerShell, you can also install the hook explicitly:

```powershell
cue install powershell
```

Or add one line manually:

```bash
# ~/.bashrc
eval "$(cue init bash)"

# ~/.zshrc
eval "$(cue init zsh)"

# ~/.config/fish/config.fish
cue init fish | source
```

PowerShell:

```powershell
Invoke-Expression (& { (cue init powershell | Out-String) })
```

### From Source

```bash
go install github.com/laeioun/cue@latest
```

For local development:

```bash
go build -o cue .
mkdir -p ~/.local/bin
cp cue ~/.local/bin/
eval "$(cue init bash)"
```

## Supported Shells

- bash
- fish
- zsh
- PowerShell with PSReadLine

## Supported Commands

- `git`, `gh`, `cargo`, and `docker` from bundled YAML specs
- User specs from `~/.config/cue/specs/<command>.yaml`
- Other commands through a best-effort `--help`, `-h`, then `help` fallback parser

Fallback completions are cached under your user cache directory, usually `~/.cache/cue`.

## Adding A Spec

Generate a draft spec from any command with parseable help:

```bash
cue spec generate cargo
cue spec generate myapp
```

Generated specs are written to `~/.config/cue/specs/<command>.yaml`, where you can edit them by hand. Use `--force` to replace an existing generated spec.

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
