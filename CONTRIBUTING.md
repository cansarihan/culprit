# Contributing

Culprit is a single Go binary with no dependencies outside the standard library.

## Adding a signal

Signals live in `internal/signals`, one file per family: what the message says,
what the diff adds, and which files it touches. Define the signal with a weight
and a one-line description, then add a case to `signals_test.go` that shows it
firing on the shape it is meant to catch and staying quiet on a near miss.

If a signal changes how the demo history scores, update the expectations in
`internal/scan/scan_test.go` too — they pin the four silent fixes and the one
announced fix that the demo hides.

## Before a change

- `make lint` and `make test` must pass. Tests need `git` on the path.
- `gofmt` is enforced; run `make fmt`.

## Commits

Conventional commits (`feat:`, `fix:`, `test:`, `docs:`, `build:`, `ci:`), one
logical change each.
