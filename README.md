# blink

A hot-reload style CLI tool for World of Warcraft addon developers. Watches your addon source files for changes and automatically syncs them to the WoW AddOns folder — like nodemon, but for WoW.

## Demo

![blink demo](assets/demo.gif)

## Features

- **File watching** — Detects changes via OS-level events and copies files instantly
- **Auto-detect addon source** — Finds your addon by scanning for `.toc` files, or specify a path manually
- **Smart ignore** — Respects `.gitignore` and `.pkgmeta` ignore lists automatically, with additional patterns via config
- **Deletion sync** — Target mirrors source exactly; removed source files are cleaned up
- **Polished TUI** — Spinner, status header, and rolling change log; falls back to plain text when piped

## Install

### From source

```bash
go install github.com/byteorem/blink/cmd/blink@latest
```

### Build from repo

```bash
git clone https://github.com/byteorem/blink.git
cd blink
go build ./cmd/blink
```

## Quick Start

Set your WoW path in `blink.toml` and run `blink` from your addon project directory:

```toml
# blink.toml
wowPath = "/mnt/c/Program Files/World of Warcraft/_retail_"
```

```bash
cd ~/my-addon
blink
```

Blink finds the `.toc` file, copies everything to your WoW AddOns folder, and watches for changes.

## Usage

```
blink [flags]

Flags:
  --source, -s      Path to addon source (default: auto-detect via .toc files)
  --wow-path, -w    Path to WoW version folder, e.g. /path/to/WoW/_retail_ (required)
  --no-watch        One-time copy, don't watch for changes
  --version, -v     Print the version
```

```bash
# Specify a custom WoW path
blink --source ./MyAddon --wow-path "C:\Program Files\World of Warcraft\_retail_"

# One-time copy without watching
blink --no-watch
```

## Configuration

Blink can be configured via a `blink.toml` file in your project root:

```toml
source = "./MyAddon"
wowPath = "C:\\Program Files\\World of Warcraft\\_retail_"
ignore = ["*.md", "tests/"]
useGitignore = true
usePkgMeta = true
```

| Field          | Description                                              | Default    |
|----------------|----------------------------------------------------------|------------|
| `source`       | Path to addon source, or auto-detect via `.toc` files    | `"auto"`   |
| `wowPath`      | Path to WoW version folder (e.g. `.../_retail_`) — **required** | —        |
| `ignore`       | Additional glob patterns to ignore (on top of .gitignore)| `[]`       |
| `useGitignore` | Respect `.gitignore` patterns                            | `true`     |
| `usePkgMeta`   | Respect `.pkgmeta` ignore patterns                       | `true`     |

**Precedence**: CLI flags > `blink.toml` > defaults

> **Note**: Blink accepts both Windows paths (`C:\...`) and WSL-style paths (`/mnt/c/...`).

See [`blink.toml.example`](blink.toml.example) for a commented template.

### Ignore strategy

1. `.git/` and `blink.toml` are always ignored
2. `.gitignore` patterns are respected automatically (disable with `useGitignore = false`)
3. `.pkgmeta` ignore list is respected automatically (disable with `usePkgMeta = false`)
4. Additional patterns from the `ignore` config array

## Requirements

- Go 1.21+
- Windows (WSL-style paths like `/mnt/c/...` are also supported)

## License

MIT
