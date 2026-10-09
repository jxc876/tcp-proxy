# AGENTS.md

## Project overview

This repository implements a TCP proxy in Go. 

The focus is on simplicity for learning purposes.

We don't need production grade code if it overcomplicates things.


## Important directories

- `cmd/`: executable entry points
- `internal/proxy/`: connection and forwarding logic
- `internal/config/`: configuration parsing
- `integration/`: end-to-end tests

Additionally:
- `notes/`: my personal study notes, you can skip these

## Development commands

- Build: `go build ./...`
- Unit tests: `go test ./...`
- Formatting: `gofmt -w <changed-go-files>`
- Static analysis: `go vet ./...`

## Coding conventions

- Prefer standard-library packages, unless a dependency provides substantial value
- Return errors with useful context
- Keep exported APIs documented
- Avoid unrelated refactoring in focused changes

## Verification

After changing proxy behavior:

1. Run `go test ./...`.
2. Run the relevant integration tests.
3. Add or update tests for externally observable behavior.

## Generated files

Do not edit generated files directly. 

Update their source and rerun the corresponding generator.
