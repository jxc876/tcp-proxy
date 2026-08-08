# Go Modules, Packages, and Binaries

Go uses several related names that serve different purposes:

- A **module** is a collection of related packages that are versioned together
- A **package** is a directory containing Go source files compiled together
- A **command** is a package named `main` that can be built into an executable
- A **binary** is the executable file produced by building a command

These names do not have to be the same.

## Module paths

The `module` directive in the root `go.mod` file declares the module's identity:

```go
// go.mod
module tcp-load-balancer
```

The module path is also the prefix used to import packages within the module. In
this project, the package in `internal/proxy` is therefore imported as:

```go
// cmd/load-balancer/main.go
import "tcp-load-balancer/internal/proxy"
```

The module path is not the name of a directory that must contain Go files, and it
does not determine the name of a generated binary.

## Repo-Based Module

For a module that may be published or imported by another project, its path should
normally identify the source repository. 

For this project, the conventional path would be:

```go
module github.com/jxc876/tcp-proxy
       └───── module path ───────┘
```

A repository-based module path is composed of:
* github.com — hosting domain
* jxc876 — repository owner
* tcp-proxy — repository name


Its internal package would then be imported as:

```go
import "github.com/jxc876/tcp-proxy/internal/proxy"
```

A short path such as `tcp-load-balancer` can work for a self-contained local
module, but a repository-based path avoids ambiguity and tells Go where the module
can be downloaded.

## Packages and directories

In Go, a package generally corresponds to one directory of `.go` files. 

This project contains these packages:

```text
cmd/load-balancer  -> package main; runnable command
cmd/demo-backend   -> package main; runnable command
internal/proxy     -> package proxy; reusable library code
```

The repository root contains `go.mod`, but it has no `.go` files. It is the
module root, not a package. 

A module root and a package directory can be the same directory, but they do not have to be.

## What `go run` targets

The argument to `go run` selects a package. 

A relative path is resolved from the current directory:

```bash
go run ./cmd/load-balancer
```

This compiles and runs the `main` package in `cmd/load-balancer`.

The dot means "the package in the current directory":

```bash
go run .
```

Running that command from the repository root fails because the root contains no
Go source files. It works after changing to a runnable package directory:

```bash
cd cmd/load-balancer
go run .
```

The first form is usually more convenient because it can be run from the module
root.

`go run` builds a temporary executable and immediately runs it. The temporary
binary is not left in the project for later use.

## Building a persistent binary

Use `go build` with `-o` when an executable should remain on disk:

```bash
go build -o tcp-proxy ./cmd/load-balancer
./tcp-proxy
```

Here the names have separate meanings:

```text
github.com/jxc876/tcp-proxy  -> recommended module path
./cmd/load-balancer          -> package being built
tcp-proxy                    -> output binary name chosen with -o
```

## Rules to remember

- A module groups and versions packages
- The module path is the import-path prefix for packages inside the module
- Published module paths normally match their repository location
- A package is normally one directory containing `.go` files
- Only a `package main` with a `main` function is directly runnable
- `.` means the package in the current directory, not the entire module
- `go run` creates and runs a temporary binary
- `go build -o name` creates a persistent binary with the chosen name
- A module path, package path, directory name, and binary name are related but
  distinct concepts


