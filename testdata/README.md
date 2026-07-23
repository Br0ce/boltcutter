# testdata

Golden fixtures for the test suite. Files here are ignored by the Go
build tool and are committed deliberately as reviewed artifacts.

## test.db

A small, deterministic bbolt database used to exercise reading of a real
on-disk bbolt file. Contents:

- `users` — five key/value pairs (`user:001`…`user:005`) with JSON values
- `config` — `version`, `enabled`, and a nested `flags` sub-bucket

Regenerate after changing the generator:

```
go generate ./...
```

Produced with **bbolt v1.5.0**. If you bump the bbolt dependency,
regenerate and review the resulting binary change.
