![Build Status](https://github.com/Br0ce/boltcutter/actions/workflows/ci.yml/badge.svg)
[![go.mod Go version](https://img.shields.io/github/go-mod/go-version/Br0ce/boltcutter)](https://github.com/Br0ce/boltcutter)
[![Go Reference](https://pkg.go.dev/badge/github.com/Br0ce/boltcutter.svg)](https://pkg.go.dev/github.com/Br0ce/boltcutter)

# BoltCutter

A terminal UI for browsing [bbolt](https://github.com/etcd-io/bbolt) databases.

BoltCutter opens a bbolt database file and lets you navigate its buckets and key/value pairs interactively — no need to write ad-hoc scripts to peek inside.

## Usage

```
boltcutter path/to/my.db
```

The database is opened read-only. A header names the bucket that is open
on the left, and the database file and its size on the right; the
shortcuts sit in a footer below. Three panes fill the screen between
them. The listings fill the panes from the left: at the root the cursor
is in the left pane, and once you dive into a bucket the left pane keeps
the bucket you came from while the cursor moves to the middle and stays
there, however deep you go. The right pane is reserved for the value of
the selected key. Buckets are marked with a trailing `/`.

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
