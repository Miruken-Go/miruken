# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
go build ./...                          # Build all packages
go test ./... -count=1                  # Run all tests (no cache)
go test ./test/ -count=1                # Run core tests only
go test ./test/ -run TestHandlesTestSuite/TestHandles  # Run single test
go test ./test/ -bench=. -benchmem      # Run benchmarks
golangci-lint run                       # Lint (see .golangci.yml)
```

Tests use **testify/suite** — each package has a `test/` subdirectory with suite-based tests. Benchmarks live in separate `*_bench_test.go` files.

## Architecture

Miruken is a **handler-based callback dispatching and dependency injection framework** built on Go reflection. The core abstraction is `Handler.Handle(callback, greedy, composer)` which dispatches callbacks through type-variance matching.

### Dispatch Flow

1. A callback (e.g. `Handles`, `Provides`, `Creates`) enters `Handler.Handle()`
2. `HandlerRuntime` (created at setup time via reflection introspection of handler types) looks up `Binding`s matching the callback's `Policy`
3. The `Policy` uses variance rules to match the callback key against binding keys
4. Matched bindings execute through a **filter pipeline** (ordered by `FilterStage`)
5. Arguments are resolved via `funcCall.resolveArgs()` which recursively dispatches dependency callbacks
6. Results flow back through `CallbackBase.ReceiveResult()`

### Variance Policies

| Policy | File | Matches | Used By |
|--------|------|---------|---------|
| **Covariant** | `covar.go` | Output type assignable to key | `provides/` — dependency resolution |
| **Contravariant** | `contravar.go` | Key assignable to input type | `handles/` — command/event handling |
| **Bivariant** | `bivar.go` | Both input and output via `DiKey` | Maps/transforms |
| **Invariant** | `invar.go` | Exact key match only | Strict lookups |

### Key Abstractions

- **`Handler`** — dispatches callbacks; composed via `AddHandlers`, `BuildUp`
- **`Callback`** — carries key, source, target, policy, constraints
- **`Policy`** — variance-based binding lookup (`MatchesKey`)
- **`Binding`** — connects a callback to a handler method via `funcCall`
- **`funcCall`** — core invocation unit; caches `reflect.Type`, resolves args, calls `reflect.Value.Call`
- **`Filter`/`FilterProvider`** — middleware pipeline with ordered stages (logging=1000, auth=3000, validation=5000)
- **`context.Context`** — tree-structured scoping with lifecycle (Active→Ending→Ended)

### Setup & Registration

```go
handler, ctx := setup.New(features...).
    Specs(&MyHandler{}, &MyProvider{}).
    Context()
```

`setup.Builder` installs `Feature`s in dependency order, introspects handler types via `describe.go` to build `HandlerRuntime`s, and creates a root `context.Context`. Features implement `Feature.Install(b)` and optionally `DependsOn()`.

### Handler Method Conventions

Handler methods are discovered by reflection based on their first parameter embedding a policy marker:

- **`handles.It`** — contravariant handler (second param is the callback type)
- **`provides.It`** — covariant provider (return type is what's provided)
- **`creates.It`** — object creation
- **`Constructor()`** — zero-arg method signals the type is constructable
- **`Init*()`** — methods prefixed with Init are implicit initializers

### Lifestyles

- **Transient** — new instance per resolution (default)
- **`provides.Single`** — singleton, cached after first creation
- **Scoped** — one instance per `context.Context`

### Policy Binding Storage

`bindingMap` in `policy.go` indexes bindings using `indexedBindingList` — variant bindings keyed by `reflect.Type`, invariant bindings keyed by string. A dynamic index tracks interface relationships discovered at runtime. Copy-on-write via `atomic.Pointer` enables lock-free reads.

## Module

Go 1.26+. Module path: `github.com/miruken-go/miruken`
