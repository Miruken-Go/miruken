# Miruken: `setup` and `context` packages

Covers `github.com/miruken-go/miruken/setup` and `github.com/miruken-go/miruken/context`. For the core dispatch engine these build on (Handler/Callback/Policy/Binding/funcCall, filter pipeline, the anonymous-struct-pointer "annotation" binding-spec system, Lifestyles), see `reference/core-dispatch.md`.

## Overview

`setup` is the composition-root / bootstrapping layer: it turns a list of `Feature`s and handler `Specs` into a fully wired root `context.Context` (which is itself a `miruken.Handler`). `context` provides the runtime scope tree (`Context`) used for request-scoped/lifetime-scoped resolution (`Scoped` lifestyle) and lifecycle notifications (ending/ended, child ending/ended).

Analogy: `setup.Builder` is roughly Spring's `ApplicationContext` builder / `@Configuration` aggregation phase; `context.Context` is roughly a Spring `ApplicationContext` instance at runtime, but explicitly tree-structured (parent/child) so nested request scopes are first-class, not simulated.

## Feature & Builder

`setup/feature.go`:
- `Feature` interface: `Install(*Builder) error`. That's the entire contract — a Feature mutates the Builder (registers specs, filters, options, other builders, etc.).
- `FeatureFunc func(*Builder) error` — function adapter to satisfy `Feature`.
- `FeatureSet(features ...Feature) Feature` — bundles multiple Features into one; internally a `featureSet` whose `Install` is a no-op and whose `DependsOn()` returns the wrapped features. This is the composition mechanism for "install this whole group as one Feature."
- Optional extension point: any `Feature` can additionally implement `DependsOn() []Feature` (dependency Features to install first) and/or `AfterInstall(*Builder, *context.Context) error` (hook run once the root `Context` exists, e.g. for validation or eager wiring that needs the live context).

`setup/builder.go` — `Builder` struct and fluent methods (all return `*Builder`, chainable):
| Method | Purpose |
|---|---|
| `Features(...Feature)` | queue Features to install |
| `Handlers(...any)` | register already-constructed handler instances directly (bypass reflection inference) |
| `Specs(...any)` | register handler *types* (or instances used as type specs) to be introspected reflectively and made available via inference |
| `ExcludeSpecs(...miruken.Predicate[miruken.HandlerSpec])` | filter out specs by predicate (matched against `miruken.TypeSpec` — e.g. by `.Name()` or `.Type()`) before they're registered |
| `Filters(...miruken.FilterProvider)` | shorthand for `Builders(miruken.ProvideFilters(providers...))` — add global filters |
| `Builders(...miruken.Builder)` | add arbitrary `miruken.Builder` decorators applied to the final handler chain (`miruken.BuildUp`) |
| `With(...any)` | shorthand for `Builders(miruken.With(values...))` — inject ambient values |
| `Options(...any)` | add option values; if the value is itself a `miruken.Builder` it's added via `Builders`, else wrapped with `miruken.Options(option)` |
| `Parsers(...miruken.BindingParser)` | extra `BindingParser`s for the reflective binding-spec system (custom "annotation" parsing — see core-dispatch.md) |
| `Observers(...miruken.HandlerRuntimeObserver)` | observe `HandlerRuntime` construction |
| `Factory(func([]BindingParser, []HandlerRuntimeObserver) miruken.HandlerRuntimeFactory)` | override the default `HandlerRuntimeFactory` construction entirely |
| `WithoutInference()` | disable lazy/implicit inference handler; specs are eagerly registered via `factory.Register` instead |
| `Tag(tag any) bool` | idempotency helper — first call with a given tag returns `true`, subsequent calls `false`. Used by Features to guard against double-installation when depended-on by multiple paths (see "Installs once" behavior below) |
| `Context() (*context.Context, error)` | build the handler graph, run `bootstrapper.bootstrap` synchronously (awaits startup promises), return the root `Context` |
| `ContextAsync() *promise.Promise[*context.Context]` | async variant — returns a promise instead of blocking on Startup |

`New(features ...Feature) *Builder` is the entry point: `setup.New(features...).Specs(...).Context()`.

### Install pipeline (`installGraph` + `build`, builder.go:216-244, 153-214)

1. `installGraph(features)` does a **level-order (BFS) traversal** over a work queue seeded with the top-level Features. For each Feature dequeued: if it implements `DependsOn() []Feature`, its dependencies are pushed to the back of the queue (so they get installed, but only *after* the current level — this lets later/explicit Features effectively override earlier-queued dependency defaults since Specs/Builders accumulate and later registrations can exclude/override). Then `feature.Install(s)` runs. Errors from all Features are joined (`errors.Join`), not short-circuited — the whole graph is attempted before reporting failure.
   - Because `Feature.Install` is called for every enqueued Feature instance (including duplicate dependency references), Features that must not double-register use `Builder.Tag(...)` as a guard (see `MyInstaller.Install` in setup/test/setup_test.go:80-88, and the "Installs once" test).
2. `build()`:
   - Runs `installGraph`.
   - Builds a `miruken.HandlerRuntimeFactory` (via `s.factory` override or the default `miruken.HandlerRuntimeFactoryBuilder` seeded with `s.parsers`/`s.observers`).
   - Starts a `handler miruken.Handler` chain with `&miruken.CurrentHandlerRuntimeFactoryProvider{Factory: factory}`.
   - Appends an internal `&bootstrapper{}` to the spec list (see Bootstrap below), then for every spec: converts it to a `miruken.HandlerSpec` via `factory.Spec(spec)`, drops it if `nil` or excluded by `s.exclude`; if `noInfer` is set, eagerly registers it (`factory.Register`) and panics on error, else collects it for lazy inference.
   - If any specs remain for inference, wraps the handler chain with `miruken.NewInferenceHandler(factory, hs)` (this is what makes handler resolution "just work" the first time a matching callback/type is seen, without every type being eagerly constructed/registered).
   - Explicit `Handlers(...)` instances are appended via `miruken.AddHandlers` (these sit *after* the spec-based/inferred handlers in the chain and are not filtered by `ExcludeSpecs`).
   - `Builders(...)` decorators are applied via `miruken.BuildUp`.
   - `context.New(handler)` constructs the root `Context` wrapping the fully composed handler chain.
   - Every installed Feature that implements `AfterInstall(*Builder, *context.Context) error` is invoked with the finished root context; errors are joined into `buildErrors`.
   - Returns `(ctx, buildErrors)` — note errors do **not** prevent a `Context` from being returned; callers must check `err` themselves (see "Errors" test: `BadInstaller.Install` and `.AfterInstall` both error and both messages appear joined in the final error string).

### Bootstrap (`setup/bootstrap.go`)

- `Options{StartupTimeout, ShutdownTimeout time.Duration}` — passed via `.Options(setup.Options{...})`.
- `Bootstrap` interface: `Startup(ctx context.Context, h miruken.Handler) *promise.Promise[struct{}]` / `Shutdown(ctx context.Context) *promise.Promise[struct{}]`. Register instances by making them resolvable (e.g. via `Specs`) as implementations of `Bootstrap` — they're collected as `[]Bootstrap` dependency.
- `bootstrapper` is an internal `Scoped` (rooted in the context — `provides.It` + `context.Scoped`) singleton-like handler constructed via reflection (`Constructor` method takes `[]Bootstrap` and `Options` via `args.Optional`/`args.FromOptions`).
- `Builder.Context()` calls `miruken.Resolve[*bootstrapper](ctx)` after building, then `.bootstrap(ctx)` which runs **all `Startup` calls concurrently** (`promise.All`), honoring `StartupTimeout` via context cancellation. `Context()` blocks (`.Await()`) on this; `ContextAsync()` returns the promise instead.
- `bootstrapper.Dispose()` (invoked when its owning Context ends, standard `Disposable` convention) runs `Shutdown` for all bootstraps **in reverse registration order**, sequentially awaited together, honoring `ShutdownTimeout`; a failure **panics** (`"failed to gracefully shutdown"`).

## Context Lifecycle (`context/context.go`)

`Context` embeds `miruken.MutableHandlers` (so it *is* a mutable `Handler` chain) plus `parent *Context`, `state State`, a thread-safe `children` list, and an observer table (copy-on-write via `atomic.Pointer`).

**States** — exactly three, linear, one-way:
```
StateActive → StateEnding → StateEnded
```
(`State` is `uint`; constants `StateActive=0, StateEnding=1, StateEnded=2`.) `ensureActive()` panics `"the context has already ended"` if any active-only operation (`NewChild`, `Observe`) is attempted outside `StateActive`.

- `New(handlers ...any) *Context` — root context constructor; always self-registers a `miruken.NewProvider(context)` so the context can resolve itself (`*Context`) as a dependency.
- `NewChild() *Context` — creates a child, wires it with a fresh handler chain seeded by `miruken.NewProvider(child)`, and subscribes two observers on the child so the **parent** is notified (`contextChildEndingObserver`/`contextChildEndedObserver`) when the child ends; on child-ended the parent also removes it from `children`. Children tracked in a `slices2.Safe[miruken.Traversing]` (concurrent-safe slice).
- `Handle(callback, greedy, composer)` — dispatches on itself first (`MutableHandlers.Handle`); **only if `greedy` is true** and unhandled, falls back to `c.parent.Handle(...)`. Non-greedy dispatch never bubbles to the parent — a context only implicitly "inherits" parent handlers when the caller explicitly requests greedy dispatch. `composer` defaults to `&miruken.CompositionScope{Handler: c}` if nil.
- `HandleAxis(axis, callback, greedy, composer)` — dispatches along a `miruken.TraversingAxis` (self/children/etc., see `axis.go` in root package) using `miruken.TraverseAxis` + a `TraversalVisitorFunc`; stops early once `result.Stop()` or `(handled && !greedy)`.
- `End(reason any)` — no-op if not `StateActive`. Sets `StateEnding`, notifies `EndingObserver`s, `defer`s setting `StateEnded` + notifying `EndedObserver`s, and in between calls `Unwind(nil)`.
- `Unwind(reason any)` — ends all children **in reverse order** (LIFO — most-recently-added child ends first), each child's own `End` cascades recursively. `UnwindToRoot(reason)` walks to `Root()` first, then unwinds from there (so an entire tree collapses top-down-by-recursion / children-first).
- `Dispose()` → `End(ReasonDisposed)`. This is the standard `miruken.Disposable` hookup — anything holding a `*Context` and wanting to release it calls `Dispose()`.
- `Root() *Context` — walks `Parent()` until nil.
- Reasons: `ReasonAlreadyEnded`, `ReasonUnwinded` (default when `End(nil)` is called), `ReasonDisposed`.
- Observer plumbing is bitmask-based (`contextObserverType`/`contextualObserverType`), supports `EndingObserver`, `EndedObserver`, `ChildEndingObserver`, `ChildEndedObserver` (on `Context`) and `ChangingObserver`/`ChangedObserver` (on `ContextualBase`, fired when a `Contextual`'s owning context is swapped). `Observe(observer)` inspects which interfaces `observer` implements to pick the right bitmask, and calling it when already in a terminal state immediately fires the observer synchronously with `ReasonAlreadyEnded` (e.g. subscribing `EndingObserver` on an already-`StateEnding` context fires immediately) rather than silently missing the transition.
- **`Contextual`** interface (`Context() *Context`, `SetContext(*Context)`, `Observe(Observer) Disposable`) + **`ContextualBase`** struct — mixin for domain objects that are *owned by* a context (as opposed to *being* one). `ChangeContext` removes the object as a handler from the old context, adds it to the new one (`InsertHandlers(0, contextual)` — inserted at front, so a contextual object's own handling takes priority), and fires Changing/Changed observers around the swap.
- `PublishFromRoot miruken.BuilderFunc` — a ready-made `Builder` that resolves the current `*Context`, walks to `.Root()`, and rewires the handler with `miruken.Publish.BuildUp(...)` targeting the root — i.e. "publish this event starting from the top of the context tree" as a composable option (pass to `setup.Builder.Builders(...)` or similar).

Context does **not** hold a `context.Context` (stdlib) reference for cancellation — that's a separate Go stdlib type imported under the alias in `bootstrap.go`; don't confuse `miruken/context.Context` (the scope tree node) with stdlib `context.Context` (cancellation token) — both are used side by side in `setup/bootstrap.go`.

## Scoped Lifestyle Integration (`context/lifestyle.go`)

`Scoped` is a `miruken.LifestyleProvider` (a `FilterProvider`/annotation type embeddable in the binding-spec anonymous struct, e.g. `provides.It; context2.Scoped` — see `setup/bootstrap.go:43-47` for a real example) that gives "one instance per `Context`" semantics, i.e. Miruken's equivalent of a request/session-scoped Spring bean.

- Struct tag `mode:"covariant,rooted"` (parsed via `InitWithTag`, comma-separated):
  - `rooted` → resolutions always bind to `context.Root()` regardless of which descendant context triggered resolution (see `Rooted`, a `BindingGroup` bundling `Scoped `mode:"rooted"`` — a reusable named "annotation" combining the marker + its own tag, embeddable as `context2.Rooted` in place of `context2.Scoped`).
  - `covariant` → forces the covariant-cache filter (`scopedCovar`) even when the binding key isn't `any`.
- `InitLifestyle(binding)` picks the filter implementation once per binding: `scopedCovar` if `covar` is set OR the binding's key is the `any` interface (`internal.IsAny(typ)` — i.e. the provider can satisfy many different requested types, so caching must be keyed by requested type, not just by context); otherwise the simpler `scoped` filter (single cached instance per context, no per-key distinction).
- `scoped` (single-key cache): per-`*Context` `scopedEntry{instance []any, once *sync.Once}` in a copy-on-write `map[*Context]*scopedEntry` under `atomic.Pointer`. First resolution in a given context runs the real construction (`next.Pipe()`, awaiting a promise if async) inside `sync.Once`; subsequent resolutions in the same context return the cached `instance`. If construction fails, the `sync.Once` is replaced so a later attempt can retry (failed constructions aren't permanently cached).
- `scopedCovar` (multi-key cache): `map[*Context]map[key]*scopedEntry` — same idea but keyed by both context *and* the requested key (`ctx.Callback.(*provides.It).Key()`), because a covariant provider can be asked for different assignable types and each needs its own cached instance. It also opportunistically reuses an existing cached instance for a *different* key in the same context if that instance's runtime type is assignable to the newly requested type (`reflect.TypeOf(o).AssignableTo(typ)`), avoiding duplicate construction when the same object legitimately satisfies multiple interfaces.
- Both filters register for context-end cleanup: once the owning `*Context` reaches `StateEnded` (`context.Observe(EndedObserverFunc(...))`), the cached entry is evicted and, if the instance implements `miruken.Disposable`, `Dispose()` is called. If the cached instance is itself a `Contextual`, it's wired with `SetContext(context)` and observed for `ContextChanging` — and **it panics** (`"managed instances cannot change context"`) if code tries to move a scoped instance to a different non-nil context; scoped instances are pinned to the context that created them until that context ends.
- `getContext(key, ctx, provider)` decides *which* context to scope into: it refuses to scope resolution of `*Context` itself (`key == contextType` — can't cache "the context" contextually, that would be circular), checks `isCompatibleWithParent` (a nested `Scoped` resolution triggered from within another `Scoped` binding's construction is rejected unless the parent was also `rooted`-compatible — prevents a rooted scope from being silently narrowed by an inner non-rooted one, or vice versa), resolves the ambient `*Context` via `provides.Type[*Context](ctx)`, errors if it's found but `!= StateActive` (`ErrScopeInactiveContext`), and redirects to `.Root()` if `rooted`.

## Public API Surface

**setup**: `New`, `Builder` (all fluent methods above), `Feature`, `FeatureFunc`, `FeatureSet`, `Bootstrap`, `Options`.

**context**: `Context` (`New`, `NewChild`, `Store`, `Handle`, `HandleAxis`, `Observe`, `Traverse`, `Unwind`, `UnwindToRoot`, `End`, `Dispose`, `Parent`, `Root`, `Children`, `HasChildren`, `State`), `State`/`StateActive`/`StateEnding`/`StateEnded`, `Reason`/`ReasonAlreadyEnded`/`ReasonUnwinded`/`ReasonDisposed`, `Contextual`, `ContextualBase`, `Observer` family (`EndingObserver`, `EndedObserver`, `ChildEndingObserver`, `ChildEndedObserver`, `ChangingObserver`, `ChangedObserver`, and their `...Func` adapters), `PublishFromRoot`; and from lifestyle.go: `Scoped`, `Rooted`, `ErrScopeInactiveContext`.

## Common Usage Pattern

```go
ctx, err := setup.New(
        httpsrv.Feature(...),   // example: a real Feature from api/http/httpsrv
        es.Feature(...),        // example: a real Feature from es
    ).
    Specs(&MyHandler{}, &MyRepository{}). // reflectively-introspected handler types
    Options(setup.Options{StartupTimeout: 30 * time.Second}).
    Context()
if err != nil {
    // installation or bootstrap failed — ctx may still be non-nil, check err explicitly
}
defer ctx.End(nil)

result := ctx.Handle(&SomeCommand{}, false, nil)          // dispatch a callback directly
v, promise, ok, err := miruken.Resolve[*MyRepository](ctx) // or resolve a dependency
```

Confirmed against `setup/test/setup_test.go`:
- `setup.New(TestFeature).Context()` (line 122) — minimal case, a package-level `Feature` var/func.
- `setup.New(TestFeature).ExcludeSpecs(func(spec miruken.HandlerSpec) bool {...}).Context()` (line 135) — filtering out specific inferred types by name or exact `reflect.Type`.
- `setup.New(TestFeature).WithoutInference().Context()` (line 165) — disables implicit inference; only eagerly-registered specs resolve.
- `setup.New(&RootInstaller{}).Context()` (line 192) with `RootInstaller.DependsOn()` returning `[]setup.Feature{&MyInstaller{}}` — dependency Features are installed even though not passed directly to `New`.
- `setup.New(installer, installer).Context()` (line 182) plus `Builder.Tag(...)` inside `MyInstaller.Install` — demonstrates the idempotent-install guard pattern for Features that might be reached twice (once directly, once via another Feature's `DependsOn`).
- `setup.New(TestFeature).Options(setup.Options{StartupTimeout: time.Millisecond}).Context()` (line 221) returning a `promise.CanceledError` when a `Bootstrap.Startup` exceeds `StartupTimeout` — `Context()` returns `(nil, err)` in that failure case.

Every child scope (e.g. a per-request context in an HTTP handler) is created with `rootOrParentCtx.NewChild()`, used for the duration of the request, then ended with `child.End(nil)` (or `Dispose()`), which cascades to evict any `Scoped` instances created within it and disposes them.

## Open questions / uncertainties

- I did not find the code that actually calls `Context.NewChild()` per HTTP request (that likely lives in `api/http/httpsrv/handler.go` or `pipeline.go`, out of scope for this file) — confirm there if you need the exact per-request scoping call site.
- `miruken.TraverseAxis`, `miruken.CompositionScope`, `miruken.NewProvider`, `miruken.MutableHandlers`, `miruken.LifestyleProvider`, `miruken.Lifestyle`, `provides.Type[T]`, and the `HandlerRuntimeFactory`/`HandlerSpec`/`TypeSpec`/inference-handler machinery are all root-package concepts `setup`/`context` consume but don't define — they should already be covered in `reference/core-dispatch.md`; if any are missing there, this file's descriptions of `Builder.build()` and the `Scoped` filters are the best secondary source.
- `slices2.Safe[T]` (internal/slices) is a small concurrency helper (`Items()`, `Append()`, `Delete(predicate)`) — not detailed here since it's a generic internal utility, not architecture.
