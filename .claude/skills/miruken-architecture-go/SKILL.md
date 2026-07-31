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
- **CORRECTED (2026-07-31, Simplify Pass): the "3 duplicate next()-closure chains" are NOT a
  clean unification candidate.** Actually compared the three closure signatures (core filter
  pipeline, `http.Router.invoke`, `httpsrv.Pipe`): 3 params/3 returns, 1 param/0 returns
  (side-effecting, writes to `http.ResponseWriter`), 0 params/2 returns respectively —
  genuinely different shapes per domain, not one repeated pattern. Unifying would need
  `any`-boxing or reflection, trading real type safety for marginal line savings — a
  regression. Left as-is; this replaces the earlier, shallower note that flagged it as a
  consolidation candidate.
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
- **DONE (2026-07-31, Module Organization + Simplify Passes): module boundaries.** Root module
  (`github.com/miruken-go/miruken`) now has **six** split-out submodules. Each requires root at
  the first tag that actually excludes it — a nested `go.mod` inside a subdirectory only takes
  effect for consumers pinned to a root version tagged *after* the exclusion exists (see
  `es/goes`/`openapi`'s history for the original precedent):
  - `api/http/httpsrv/openapi` (`kin-openapi`/schema-gen deps — pre-existing) — root `v0.32.0`
  - `es/goes` (`modernice/goes` + transitives — pre-existing) — root `v0.32.0`
  - `security/jwt` (incl. `security/jwt/jwks`) — `golang-jwt/jwt`, `MicahParks/keyfunc`,
    `MicahParks/jwkset` — root `v0.32.1`
  - `validates/play` — `go-playground/validator`, `universal-translator`, `locales` — root `v0.32.1`
  - `validates/go` — `asaskevich/govalidator` — root `v0.32.1`
  - `config/koanf` — `knadh/koanf/*` family — root `v0.32.2`

  **Vendor folder**: investigated, recommended against — no evidence of offline/air-gapped
  build needs, `go.sum` already covers supply-chain integrity. No action taken.

  **`internal/` usage**: reviewed every top-level package for "this is really just plumbing"
  candidates (`constraint/`, `args/`, `effect/`, `cascade/`, `either/`) — all are intentional,
  consumer-facing thin façade packages (same shape as `handles/`/`provides/`/`creates/`), just
  low-usage. No misplaced package found; current `internal/`/`internal/seq`/`internal/slices`
  scoping is correct as-is. No action taken.

  **Declined splits**: `logs/` (`go-logr` is already a direct `httpsrv` dependency regardless,
  and near-zero-weight anyway — splitting wouldn't reduce a real consumer's footprint) and
  `api/json/stdjson` (`conjson` is light, and `stdjson` is the framework's default JSON
  backend, not an optional add-on).

  **`config/koanf` split — done (2026-07-31, Simplify Pass).** Was blocked because
  `security/login/test` and `security/password/test` imported it directly; fixed by reworking
  both: `security/password/test/login_test.go` now builds a `login.Flow` directly via
  `login.NewFlow(...)` instead of round-tripping through koanf (the password module never
  knew or cared how its options map was built, so zero coverage loss; collapsed what had been
  a redundant JSON-vs-ENV split into one test now that config isn't in the picture).
  `security/login/test/login_test.go`'s `Configuration/File` and `Configuration/Env` sub-tests
  were deleted outright as redundant with `config/koanf/test/provider_test.go` (which already
  covers those exact koanf behaviors); `Configuration/No Modules` was kept (it has real,
  distinct value — proves `login.Context.initFlow` propagates a config-resolved `Flow`'s
  `Validate()` failure into a `login.Error`) with a trivial in-file `emptyProvider` swapped in
  for the koanf backend. Then split exactly like the other five.

  **Heads up**: local downstream consumers in the shared workspace
  (`demo.microservice/adb2c`, `team`, `team-srv`) currently import `security/jwt`/
  `validates/play`/`validates/go` "for free" via root — they'll need `go mod tidy` to pick up
  explicit requires on the new module paths (the shared `go.work` keeps them building locally
  regardless).

- **DONE (2026-07-31, Simplify Pass): dead code and modernization.** Deleted `pipeline()` in
  `filter.go` — byte-for-byte identical to `pipelineInvoke` except it took the terminal step
  as a parameter, confirmed via repo-wide grep to have zero call sites anywhere. Replaced two
  manual "clone a map minus one key" loops in `context/lifestyle.go` (`ContextChanging`,
  `removeContext`) with `maps.Clone`+`delete`, matching the idiom the file already used
  elsewhere. Replaced two `sort.Slice`/`sort.Strings` calls (`api/http/httpsrv/openapi/feature.go`,
  `validates/outcome.go`) with `slices.SortFunc`/`slices.Sort`. All zero-behavior-change. A
  broader Go-1.26-modernization scan otherwise came back clean: no literal `interface{}`
  anywhere, `iter.Seq`/`iter.Seq2` already used idiomatically and extensively, no manual
  min/max patterns found.
- **DONE (2026-07-31, Simplify Pass, BREAKING root API change): `graph.go` rewritten around
  `iter.Seq[Traversing]`.** Initially declined as "a real rewrite, not a cheap win" — revisited
  on request and it turned out to be a genuine improvement, not just a style change: callers
  now write a plain `for node := range TraversePreOrder(root) { ... break ... }` instead of
  implementing `TraversalVisitor`/wrapping a func in `TraversalVisitorFunc` (both removed).
  Circularity detection now panics instead of returning an error (confirmed nothing anywhere
  caught it specially first); `context.Context.HandleAxis` gets a targeted `recover()` to keep
  its own external contract unchanged. Zero usage found in local demo-consumer repos, so no
  known real-world breakage. Full detail, including a minor stop-early correctness fix that
  fell out of the redesign, in `core-dispatch.md`'s graph.go entry.
- **DONE + ONE NEAR-MISS (2026-07-31, iter.Seq follow-up pass).** Surveyed the whole repo for
  more `iter.Seq`/`iter.Pull` candidates after `graph.go`. Converted `api/http/router.go`'s
  `Router.invoke` and `api/http/httpsrv/pipeline.go`'s `Pipe` to `iter.Pull(slices.Values(...))`
  — verified safe first by reading every implementation (no promise anywhere in
  `Policy.Apply`/`Middleware.ServeHTTP`'s signatures; async DI resolution blocks via `.Await()`
  rather than deferring). Swapped two plain manual reverse-index loops (`build.go`'s
  `PipeBuilders`, `context/context.go`'s `Unwind`) for `slices.Backward` — zero risk, no
  promises involved at all. **Did NOT touch `filter.go`'s `pipelineInvoke`/
  `filterBindingGroup.invoke`** — a first attempt on `pipelineInvoke` was implemented and
  reverted before being committed once it became clear its `next` continuation can genuinely
  resume on a different goroutine (async `Filter.Next` implementations exist, e.g.
  `security/authorizes/filter.go`), which violates `iter.Pull`'s documented single-goroutine
  constraint. Full trace in `core-dispatch.md` §12. **The rule going forward:** `iter.Seq` you
  write yourself (a generator, driven synchronously by whoever ranges over it) is safe
  wherever the underlying logic is genuinely synchronous end-to-end; `iter.Pull` (converting an
  existing push sequence so you can call `next()` manually) is only safe if you've traced the
  *entire* call chain and confirmed no continuation can ever be resumed asynchronously — never
  assume from surface shape alone.
- **CORRECTED (2026-07-31, Simplify Pass): don't touch `constraint/`.** Not low-value as
  earlier noted — `constraint.First[T]` is used by two production files
  (`api/multipart.go:212`, `config/factory.go:81`), plus `constraint.Named` in two test files.
  Keeping as-is; this replaces the earlier "only 2 call sites, question its need" note.
- **`effect/` now has test coverage (2026-07-31, Simplify Pass)** — `effect/test/effect_test.go`,
  covering `Group`'s aggregation (0/1/many async sub-effects, short-circuit on synchronous
  error) and `Async`'s fire-and-forget semantics. It had zero internal call sites AND zero
  tests (unlike `cascade/`, which has zero internal usage but a real passing test suite) — this
  was the one "genuinely unverified, not just unused" public API surface found. `cascade/`,
  `validates/go` vs `validates/play`'s structural similarity (the correct outcome of the
  port/adapter pattern, not a smell) — reviewed, no action.

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
