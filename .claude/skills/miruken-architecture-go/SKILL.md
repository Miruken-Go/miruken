---
name: miruken-architecture-go
description: Deep reference for the Go implementation of Miruken (github.com/miruken-go/miruken) — the handler/callback/filter dispatch engine and every framework built on it (DI, HTTP/JSON, security, validation, event sourcing). Go-specific (reflection mechanics, struct-tag parsing, go.mod submodule layout) — not applicable to Miruken ports in other languages. Load before making architectural changes, performance/reflection work, or module-layout changes in a project using this framework, or before answering non-trivial questions about how it works.
metadata:
  scope: go-specific
---

# Miruken (Go) — Architecture Reference

Miruken is a language-independent architecture built on one abstraction: **message
handlers and filters**. This Go implementation (`github.com/miruken-go/miruken`) brings a
Spring-Boot-style experience to Go — DI, HTTP/JSON binding, validation, and security are
all *Features* layered declaratively on top of a single `Handler.Handle(callback, greedy,
composer)` dispatch primitive. There is no separate "controller" or "service" concept baked
into the framework; everything is a handler method matched to a callback via a variance
`Policy` and invoked through a `Filter` pipeline.

This skill exists so future sessions don't re-derive this from source every time. It is
**Go-specific** — reflection mechanics, struct-tag parsing, `go.mod` submodule boundaries.
The author has also implemented Miruken in other languages; a separate, language-agnostic
"Miruken concepts" skill (Handler/Callback/Policy/Filter/Concepts-vs-Features as pure ideas,
no Go reflection) is planned for `~/.claude/skills/` once those other repos are analyzed —
ask the user for those repo paths if that work comes up.

## The one idea to hold before reading anything else

A handler method declares *what it handles* and *what cross-cutting concerns apply to it*
by taking, as one parameter, a pointer to an **anonymous struct** whose embedded fields are
policy markers (`handles.It`, `provides.It`, `creates.It`, ...), filters, and constraints —
and whose **struct tags are the annotation arguments**:

```go
func (p *PassThroughRouter) Pass(
    _ *struct {
        handles.It
        miruken.SkipFilters
        Routes `scheme:"pass-through"`
    }, routed Routed, composer miruken.Handler,
) (any, miruken.HandleResult) { ... }
```

The parameter is always `_` — never allocated, never touched at runtime. Its `reflect.Type`
is inspected exactly once, at setup time, and the parsed result (policy, filters,
constraints, flags) is baked into an immutable `Binding`. This is Go's answer to Java
annotations: compile-time checked, zero runtime cost, IDE-navigable, composable via a named
`BindingGroup` (a reusable "meta-annotation"). It shows up in every subsystem — HTTP routing,
auth, validation, config, event sourcing. Full mechanics: `reference/core-dispatch.md` §3.

## Reference files — load only what you need

| File | Load when you're touching / asking about |
|---|---|
| `reference/core-dispatch.md` | The engine itself: `Handler`/`Callback`/`HandleResult`, the annotation-simulation binding system (`bind.go`), `describe.go` introspection, `policy.go` binding storage, the four variance policies (Covariant/Contravariant/Bivariant/Invariant), `funcCall` reflection invocation (bind-time vs dispatch-time split), the filter pipeline, lifestyles, `Handles`/`Provides`/`Creates`. **Has a dedicated §12 reflection/performance-hot-spot list with file:line refs** — read this first for any performance task. |
| `reference/setup-context.md` | `setup.Builder`/`Feature` (composition root, dependency-ordered install, bootstrap/shutdown), `context.Context` (scope tree, lifecycle states, per-request child contexts), `Scoped` lifestyle. |
| `reference/delivery-http-json.md` | `api/` tree end to end: message envelope, polymorphism (`@type`/`@values`), routing (port/adapter split), scheduling/batching, the HTTP server pipeline (middleware, `PolyHandler`, status-code mapping), auth middleware wiring, the OpenAPI submodule, JSON mapping (`stdjson`). Includes an honest hexagonal-purity assessment with file:line citations. |
| `reference/security-validation.md` | `security/` (Subject/Principal, JAAS-style pluggable login modules, password/JWT backends, declarative `authorizes` policy) and `validates/` (the validation port + two swappable adapters — govalidator and go-playground/validator — presented as a case study in clean hexagonal design). |
| `reference/eventsourcing-support.md` | `es/` (aggregate/command/event model — **flagged as an early/incomplete sketch, not production code**, see below) plus the support libraries: `promise` (async result type), `either` (Left/Right monad), `maps` (the 4th built-in policy, bivariant type-to-type transforms), `logs`, `config` (koanf-backed), `cascade`/`effect` (side-effect model). |

Each file cites exact `file:line` locations — treat them as accurate as of this writing, but
re-verify with a grep if the surrounding code has since changed significantly.

## Flagged findings worth remembering

- **`es/` is an early design sketch, not finished, and confirmed not currently active**
  (2026-07-31). Several files contain `fmt.Println` debug stand-ins instead of real behavior
  (`es/feature.go:50`, `es/aggregate/root.go:46`, `es/command/handles.go:45`), and
  `es/goes/feature.go`'s `Install` is an empty stub — the `modernice/goes` event-store
  integration isn't wired up. The user plans to build it out in the future as new
  feature work. **Exclude `es/` from simplify/performance/reorganize passes unless the user
  explicitly asks for it** — treat any future `es/` work as finishing an unfinished design,
  not cleaning up existing behavior.
- **FIXED (2026-07-31): `authorizes.Required` no longer fails open.** `authorizes.Access`
  (the free function) still respects `Options{RequirePolicy: true}` and defaults to
  *permissive* when unhandled — that's intentional and still correct for ad-hoc `Access()`
  calls with no policy wired up at all (see the `Default`/`RequiresPolicy` tests in
  `security/authorizes/test/authorizes_test.go`). But the `authorizes.Required` **filter**
  (the declarative "this method must be authorized" marker) now always fails *closed* on an
  unhandled check, regardless of the ambient `Options` — added an unexported `access(handler,
  action, requirePolicy bool, constraints...)` in `security/authorizes/it.go` that both
  `Access` (passes the configured option) and `filter.Authorize` in `filter.go` (always passes
  `true`) delegate to. Covered by a new test, `Filter/DeniedWithoutPolicy`, using a
  `Required`-protected `Account.Withdraw` method with no corresponding `AuthorizeX` policy
  registered anywhere — asserts `*authorizes.AccessDeniedError`, not silent allow. Full test
  suite (all 3 modules) passes; `go vet ./...` clean.
- **CONFIRMED, KEEP AS-IS: implicit `Provides` registration defaults to `Single` (singleton),
  not transient.** Any constructable type with no explicit `Provides` binding spec gets
  `&Single{}` force-attached (`provides.go:287-304`, `core-dispatch.md` §9). User reviewed and
  decided not to change this default — do not revisit unless asked.
- **`validates/` is the cleanest hexagonal example in the repo** — the port has zero imports
  of either backend adapter, both adapters are fully swappable with no consumer changes. Good
  reference case if the user asks "show me what clean looks like here."
- **Three structurally-identical "linked `next()`-closure" middleware chains exist**
  independently: the core filter pipeline (`filter.go`), `http.Router.invoke`, and
  `httpsrv.Pipe` (`delivery-http-json.md` §8). Candidate for consolidation in a simplify pass.
- **FIXED (2026-07-31, Performance Pass #1): `promise.Then`/`Catch` no longer always spawn a
  goroutine + channel.** They now fast-path (run synchronously on the calling goroutine) when
  the source promise is already permanently settled (`p.ch == nil` — true for
  `Resolve`/`Reject`-built promises and, critically, everything from `promise.Lift`, which the
  core dispatch engine produces on every synchronously-available dependency typed as a
  promise). `New` itself is unchanged; the genuinely-async path through `Then`/`Catch` is
  byte-for-byte identical to before. Benchmarked ~3.5x faster / 2 fewer allocs for the
  already-resolved case, no change for the async case. See `core-dispatch.md` §12 for full
  detail and the correctness review that preceded the change.
- **FIXED (2026-07-31, Performance Pass #1): `MethodBinding.Invoke`'s initArgs shuffle.**
  Replaced an `append`+self-copy with a single `make`+forward-copy — same allocation count,
  half the copy work, zero behavior change. See `core-dispatch.md` §12.
- **INVESTIGATED AND DECLINED (2026-07-31, Performance Pass #1): `orderFilters` per-dispatch
  recompute/sort.** Looked like a caching win but isn't safe to cache blindly — at least one
  `FilterProvider` (`filterSpecProvider`) does a live DI resolve per call specifically so
  `Scoped`-lifestyle filters bind to the correct ambient context, and this is the
  security/validation filter pipeline. Left untouched this round; see `core-dispatch.md` §12
  for what a safe narrower design would need to account for before revisiting.
- **Module boundaries today**: root module (`github.com/miruken-go/miruken`), plus two split-out
  submodules: `api/http/httpsrv/openapi` (keeps `kin-openapi`/schema-gen deps out of core —
  deliberate, correctly drawn per `delivery-http-json.md` §6) and `es/goes` (keeps
  `modernice/goes` + its transitive deps opt-in — `eventsourcing-support.md` §3). Relevant
  starting point for the "do I need more submodules / internal/ packages" question.

## Open items flagged by the research (not yet resolved)

- `internal/` package internals (`CopySliceIndirect`, `CopyIndirect`, `Exported`,
  `TargetValue`, `internal/seq`, `internal/slices`) were not deep-dived — referenced
  throughout but treated as a "runtime/reflection helpers" layer. Worth a dedicated pass if a
  performance task reaches into `internal/runtime.go`.
- The exact per-HTTP-request call site for `context.Context.NewChild()` wasn't pinned down
  (likely `api/http/httpsrv/handler.go:212`'s `baseHandler.serve` per `delivery-http-json.md`
  §4, which matches `setup-context.md`'s expectation) — treat as confirmed via that
  cross-reference, not independently re-verified.
- `security/jwt/jwks` (JWKS fetch/cache mechanics) was confirmed only at the port-shape level
  (`KeySet.At`/`From`), not read in depth.
