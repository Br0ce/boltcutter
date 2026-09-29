# testdata

Golden fixtures for the test suite. Files here are ignored by the Go
build tool and are committed deliberately as reviewed artifacts.

## test.db

A deterministic bbolt database, used both by the tests that read a real
on-disk file and by hand, to point the browser at something with enough
in it to be worth browsing:

    go run ./cmd testdata/test.db

Contents:

- `users` — five key/value pairs (`user:001`…`user:005`) with JSON values
- `config` — `version`, `enabled`, and a nested `flags` sub-bucket
- `events` — 50,000 key/value pairs (`event:00000`…) with JSON values.
  Far longer than any pane, so scrolling it, and jumping to its end,
  shows whether a listing costs what it shows or what it holds
- `blobs` — the awkward values: one too long for the preview to read
  whole, one that is no text at all, one that is plain text rather than
  JSON, and one whose key holds a newline and an escape sequence
- `deep` — buckets nested eight levels down, for watching the columns
  shift on the way there and back

Regenerate after changing the generator:

```
go generate ./...
```

The size of `events` is a flag, so the fixture can be made heavier
without touching the generator:

```
go run ./internal/gen -events 200000
```

The generator compacts the database on the way out, because bbolt
doubles a file as it grows and an uncompacted fixture carries about as
much empty space as data.

Produced with **bbolt v1.5.0**. If you bump the bbolt dependency,
regenerate and review the resulting binary change.
