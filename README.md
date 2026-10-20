![Build Status](https://github.com/Br0ce/boltcutter/actions/workflows/ci.yml/badge.svg)
[![go.mod Go version](https://img.shields.io/github/go-mod/go-version/Br0ce/boltcutter)](https://github.com/Br0ce/boltcutter)
[![Go Reference](https://pkg.go.dev/badge/github.com/Br0ce/boltcutter.svg)](https://pkg.go.dev/github.com/Br0ce/boltcutter)

# BoltCutter

A terminal UI for browsing [bbolt](https://github.com/etcd-io/bbolt) databases.

BoltCutter opens a bbolt database file and lets you navigate its buckets and key/value pairs interactively — no need to write ad-hoc scripts to peek inside.

## Install

Download the binary for your platform from the [releases page](https://github.com/Br0ce/boltcutter/releases).

### macOS (Apple Silicon)

```sh
VERSION=v0.1.0  # replace with the desired version
curl -LO https://github.com/Br0ce/boltcutter/releases/download/${VERSION}/boltcutter-${VERSION}-darwin-arm64

# Verify the checksum against the value published on the releases page
shasum -a 256 boltcutter-${VERSION}-darwin-arm64

chmod +x boltcutter-${VERSION}-darwin-arm64
sudo mv boltcutter-${VERSION}-darwin-arm64 /usr/local/bin/boltcutter
boltcutter --version
```

### Linux (amd64)

```sh
VERSION=v0.1.0  # replace with the desired version
curl -LO https://github.com/Br0ce/boltcutter/releases/download/${VERSION}/boltcutter-${VERSION}-linux-amd64

# Verify the checksum against the value published on the releases page
sha256sum boltcutter-${VERSION}-linux-amd64

chmod +x boltcutter-${VERSION}-linux-amd64
sudo mv boltcutter-${VERSION}-linux-amd64 /usr/local/bin/boltcutter
boltcutter --version
```

### From source

Requires Go 1.27+.

```sh
git clone https://github.com/Br0ce/boltcutter.git
cd boltcutter
make build
```

The binary will be placed at `./bin/boltcutter`.

## Usage

```
boltcutter path/to/my.db
```

| Key                     | Action                            |
| ----------------------- | --------------------------------- |
| `↑`/`k`, `↓`/`j`        | move the cursor, scroll the value |
| `pgup`/`ctrl+b`, `pgdn`/`ctrl+f` | page up / down           |
| `home`/`g`, `end`/`G`   | jump to top / bottom              |
| `enter`/`→`/`l`         | open the selected bucket          |
| `esc`/`←`/`h`/`backspace` | back to the parent bucket       |
| `tab`                   | focus the value, to scroll it     |
| `q`/`ctrl+c`            | quit                              |

Values are shown as indented, syntax-highlighted JSON. A value in no
format we can read, or one too long to preview whole, says so in the
pane instead.

## Development

Regenerate the golden test database in `testdata/`:

```
make gen-testdata
```

## License

[Apache 2.0](LICENSE)
