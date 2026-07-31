# Event Sourcing (`es/`) and Support Libraries

## 1. Overview

This file covers two unrelated areas that share only the fact that they sit on top of the core dispatch engine (see `core-dispatch.md`):

- **Event sourcing** (`es/`, `es/aggregate`, `es/command`, `es/event`, `es/goes`): a declarative CQRS/ES layer — commands mutate an aggregate by producing events, events are applied to rebuild aggregate state. **Status: early/incomplete.** Several files contain `fmt.Println` debug statements instead of real behavior (`es/feature.go:50`, `es/aggregate/root.go:46`, `es/command/handles.go:45`), and `es/goes/feature.go`'s `Install` method body is empty (lines 16-18) — the `modernice/goes` event-store integration is stubbed but not wired up. Treat this package as a design sketch, not production-ready.
- **Support libraries**: `promise` (async result type), `either` (Left/Right monad), `maps` (a 4th built-in variance policy for type-to-type transforms), `logs`, `config`, `cascade`, `effect`, `constraint` (subpackage). These are production-quality and used throughout the rest of the framework.

## 2. Event Sourcing: Aggregate/Command/Event model

### Concepts

- **Aggregate root** (`es/aggregate/root.go`): `aggregate.Root` is a **reusable annotation bundle** (a `BindingGroup`, see `core-dispatch.md` §Annotation Simulation) that a domain type embeds to declare itself as an aggregate root:
  ```go
  type Root struct {
      miruken.BindingGroup
      provides.It
      Metadata
      loadProvider
  }
  ```
  Embedding `aggregate.Root` in a `Constructor(_ *aggregate.Root)` parameter (see `es/test/todo/list.go:58-61`) simultaneously: (a) registers the type as `provides.It` (covariant, constructable), (b) attaches `aggregate.Metadata` (reads an `aggregate:"name=..."` tag), (c) attaches a `loadProvider` filter that intercepts `Provides`/`Creates` callbacks to load the aggregate's prior state before construction completes.
- **`aggregate.Metadata`** (`es/aggregate/metadata.go`): a `string` type implementing `InitWithTag` — reads `aggregate:"name=X"` from the struct tag on the embedding field, defaults to `internal.DefaultTypeName(typ)` if absent (`es/aggregate/model.go:42-44`).
- **`aggregate.Model`** (`es/aggregate/model.go`): reflects over the aggregate struct to locate its `Id` (`uuid.UUID`) and `Version` (`int`) — either as fields tagged `aggregate:"id"` / `aggregate:"version"` (`extractAggregateIdAndVersionFields`, lines 86-168) or as `Id()/SetId()/Version()/SetVersion()` methods (`extractAggregateIdAndVersionMethods`, lines 170-217). Builds `func(any) uuid.UUID` / `func(any, uuid.UUID)` closures over `reflect.Value.Field`/`.Call` — a runtime cost paid once per aggregate type when the model is built (see `es/feature.go:54-73`), not per instance.
- **`aggregate.Options`** (`es/aggregate/options.go`): `Options{Version miruken.Option[int]}`; `aggregate.Version(n)` returns a `miruken.Builder` you pass to `miruken.BuildUp(ctx, aggregate.Version(2))` to request a specific historical version when resolving an aggregate (consumed via `miruken.GetOptions[Options](ctx)` in the `loader` filter, `root.go:44-46` — currently just printed, not implemented).
- **`command.Handles`** (`es/command/handles.go`): another `BindingGroup` — embeds `handles.It` + `command.Metadata` + `processProvider`. A method on the aggregate takes `_ *command.Handles` (or `_ *struct{ command.Handles \`command:"name=X"\` }` to override the name, see `es/test/todo/list.go:93-96`) as its policy-marker parameter, making it a contravariant command handler on that aggregate instance. The `processor` filter (`Order() int { return 10 }`) is meant to interpret the method's return value as an `event.Stream` to apply — currently it just prints the output (line 45).
- **`command.Metadata` / `command.Model`**: same tag-driven naming pattern as aggregate, keyed off `command:"name=X"`.
- **`event.Applies`** (`es/event/applies.go`): a **distinct, invariant-policy Callback** (`appliesPolicyIns = &miruken.InvariantPolicy{}`, line 103) — NOT contravariant like `Handles`. `CanInfer()/CanFilter()/CanBatch()` all return `false` (lines 40-50): event application is deliberately exact-type-match only, unfiltered, unbatched — it must hit an explicit `TaskAdded(_ *event.Applies, added TaskAdded)`-shaped method on the aggregate or fail. `event.Apply(handler, evt)` (func, lines 86-101) is the entry point, analogous to `miruken.Command` for `Handles`.
- **`event.Handles`**: `BindingGroup{ handles.It; event.Metadata }` — a lighter marker for methods that handle a raw event (not apply-to-aggregate), tagged `event:"name=X"`.
- **`event.Stream`** (`es/event/stream.go`): `type Stream []any`; `event.Append(events...)` is a fluent constructor — this is the return type of command-handling methods, e.g. `func (l *List) AddTask(_ *command.Handles, add AddTask) event.Stream`.

### Worked example: `es/test/todo/list.go`

```go
type List struct {
    Id      uuid.UUID
    Version int
    tasks   []string
    archive []string
}

func (l *List) Constructor(_ *aggregate.Root) {}

func (l *List) AddTask(_ *command.Handles, add AddTask) event.Stream {
    if l.Contains(add.Task) { return nil }
    return event.Append(TaskAdded{add.Task})
}

func (l *List) TaskAdded(_ *event.Applies, added TaskAdded) {
    l.tasks = append(l.tasks, added.Task)
}
```
Flow (intended, per the design even though the cascading isn't fully wired): a `command.Handles` method (`AddTask`) returns an `event.Stream`; a `command.Handles` method can also be renamed via tag: `CompleteTasks(_ *struct{ command.Handles \`command:"name=completeTasks"\` }, ...)`. Applying events is separately triggered — tests call `event.Apply(ctx, todo.TaskAdded{...})` directly (`es/event/test/applies_test.go:42`), not automatically from the command result; the `processor`/`loader` filters that would presumably wire "run command → get stream → apply each event → persist" are the stubbed pieces.

`Id`/`Version` are plain public fields (no `aggregate:"id"` tag needed) because `aggregate.Model.NewModel` falls back to a field literally named `Id` of type `uuid.UUID` and `Version` of kind `int` (`es/aggregate/model.go:115-131`).

### Installing

```go
setup.New(es.Feature()).Specs(&todo.List{}).Context()
```
`es.Feature()` returns an `es.Installer` (`es/feature.go:95-103`) implementing `setup.Feature`. On install it registers itself as a `setup.Builder` **observer** (`b.Observers(i)`, line 30) and reacts to `HandlerRuntimeRegistered` (called once per handler type after its `HandlerRuntime` is built) by extracting the `*miruken.CtorBinding` for `provides.It`, pulling its `*aggregate.Metadata` out of `Binding.Metadata()`, and building an `aggregate.Model` (lines 54-73) — currently only prints it (`fmt.Println("Aggregate", model)`, line 50). This is the extension point for wiring persistence: a real implementation would index these models for a repository/event-store lookup.

## 3. Event Sourcing: `goes` submodule

`es/goes/` is its **own Go module** (`es/goes/go.mod`, module `github.com/miruken-go/miruken/es/goes`) depending on `github.com/modernice/goes v0.9.0` — a third-party Go event-sourcing/CQRS toolkit (command/event buses, event store adapters). It is split out so that projects using the core `es/` abstractions are not forced to pull in `modernice/goes` and its transitive deps (protobuf, mergo, etc. — see the `// indirect` block in `es/goes/go.mod`) unless they specifically want that backend. Currently `goes.Installer.Install` (`es/goes/feature.go:15-20`) is an empty stub holding `cmdBus command.Bus` / `eventBus event.Bus` fields — the adapter wiring from Miruken's `Handles`/`Applies` callbacks to `modernice/goes`'s bus types has not been implemented yet.

## 4. Promise (`promise/`)

Miruken's own `Promise[T]` (channel + `sync.Once`-based, adapted from github.com/chebyrash/promise per the comment at `promise/promise.go:9`) is the async result type threaded through the entire dispatch pipeline (`Handler.Handle` returns synchronously but wraps async work in `*promise.Promise[[]any]`; see `core-dispatch.md`).

Core API (`promise/promise.go`):
- `New[T](ctx, executor)` — executor gets `resolve func(T)`, `reject func(error)`, `onCancel func(func())`; runs on its own goroutine; a `defer p.handlePanic()` converts panics into rejections.
- `Resolve[T](v)` / `Reject[T](err)` — synchronous, already-settled promises (no channel/goroutine needed since `p.ch == nil` short-circuits `Await`).
- `Then[A,B](p, func(A) B) *Promise[B]`, `Catch[T](p, func(error) error) *Promise[T]` — chain by creating a new promise that awaits `p` internally.
- `p.Await() (T, error)` — blocks on the channel or `ctx.Done()`, whichever first; cancels `p` if the context won.
- `All[T](ctx, ...*Promise[T]) *Promise[[]T]`, `Race[T](ctx, ...*Promise[T]) *Promise[T]`.
- `Cancel()` / `OnCancel(func())` — cooperative cancellation; `CanceledError` wraps `context.Cause`.

Reflection support (`promise/reflect.go`) — this is the part that lets *generic* dispatch code hold a promise **without knowing `T`**:
- `Reflect` interface: `Context()`, `UnderlyingType() reflect.Type`, `Then(func(any) any) *Promise[any]`, `Catch(...)`, `AwaitAny() (any, error)`. Every `*Promise[T]` implements this automatically.
- `Inspect(typ) (reflect.Type, bool)` — given a `reflect.Type`, checks if it implements `Reflect` and returns its `T` via a zero-value call to `UnderlyingType()`. Used by `funcCall`/`describe.go` machinery to recognize a handler method that returns `*promise.Promise[SomeType]` without generic instantiation.
- `Lift(typ, result) Reflect` / `CoerceType(typ, promise) Reflect` — construct a `*Promise[T]` for a `reflect.Type` at runtime via `reflect.New(typ.Elem())`, using an unexported `internal` interface (`lift`/`coerce` methods) — this is how the framework "boxes" a concretely-typed result into the promise type the call site expects, purely through reflection, without generic type parameters at the call site.
- `Coerce[T](promise Reflect) *Promise[T]` — the inverse: unbox a `Reflect` into a concrete `*Promise[T]`.
- Helpers: `Unwrap` (flatten `Promise[Promise[T]]`), `Empty()`/`RejectEmpty(err)` (`Promise[struct{}]`), `Return`/`IndirectReturn` (replace resolved value, `IndirectReturn` derefs a `*B`), `Slice`, `Erase`, `Delay`.

**FIXED (2026-07-31, Performance Pass #1):** `Then`/`Catch` used to always spawn a goroutine and allocate a channel via `New` (`promise/promise.go:41-53`), even for a promise that resolves practically instantly. They now fast-path to synchronous execution when the source promise's `p.ch == nil` (permanently settled — `Resolve`/`Reject`/`Lift`-built promises). `New` itself is unchanged; genuinely-async chains are unaffected. See `core-dispatch.md` §12 for the correctness review and benchmark numbers.

## 5. Either (`either/`)

`either/monad.go` implements a classic `Either`/`Result` monad as `Monad[L, R any] = any` (a type alias over `any`, not a real sum type — Go generics can't express a tagged union directly, so this is simulated with two internal wrapper structs `left[L]{val L}` / `right[R]{val R}` and runtime type-switches in every combinator).
- Constructors: `Left[L](v) Monad[L, any]`, `Right[R](v) Monad[any, R]`.
- Combinators: `Map`, `Apply`, `FlatMap`, `MapLeft`, `Fold(e, onLeft, onRight) A`, `Match(e, onLeft, onRight)` (side-effecting version of Fold), `Seq`.
- All combinators `panic` if given `nil` or an unrecognized underlying wrapper type (e.g. `Fold`'s `default: panic(...)`, line 107) — this is **not** a safe/total API; callers must only construct `Monad` values via `Left`/`Right`.
- Usage: `api/route.go`'s batch router (`sendBatch`) uses `either.Monad[error, any]` to represent "this individual batched message either failed or succeeded" (`api/route.go:184-199`), then `either.Fold` to route each result into either a deferred rejection or resolution.

## 6. Maps — the 4th built-in policy (`maps/`)

`maps.It` (`maps/it.go`) is a `Callback` using `miruken.BivariantPolicy` (`mapsPolicy`, line 226) — confirming the variance table in the top-level project `CLAUDE.md`. Where `Handles` is contravariant (input-only) and `Provides`/`Creates` are covariant (output-only), `Maps` binds on **both** input and output type via `miruken.DiKey{In: in, Out: out}` (`maps/it.go:36-43`): `In` is either an explicit key or `reflect.TypeOf(source)`, `Out` is the target's element type. This is the mechanism behind type-to-type transforms — most concretely JSON (de)serialization (`api/json/stdjson/mapper.go` binds `Maps` methods per type pair) but generically usable for any "convert A to B" handler method.

Entry points (mirroring `Resolve`/`Create`/`Command` shape used elsewhere):
- `maps.Out[T](handler, source, ...constraints) (T, promise, *It, error)` — map `source` to a new `T`.
- `maps.Into[T](handler, source, target *T, ...)` — map into an existing target (in place).
- `maps.Key[T](handler, key, ...)` — map by explicit key instead of inferring from `source`'s type.
- `maps.All[T](handler, sourceSlice, ...)` — map a slice element-by-element, aggregating any resulting promises via `promise.All`.

**`maps.Format`** (`maps/format.go`) is a `Constraint` (not a `FilterProvider`) representing a named format (e.g. `"json"`, `"//xml//"` for a regex) with a `Direction` (`To`/`From`) — parsed from a struct tag via `to:"..."` / `from:"..."` (`InitWithTag`, lines 73-82). Matching supports four `FormatRule`s: `Equals`, `StartsWith`, `EndsWith`, `Pattern` (regex), plus a wildcard `FormatRuleAll` (`"*"`). `Format.Satisfies` (lines 92-173) implements asymmetric matching between a required format and a candidate handler's format (e.g. a handler registered for `to:"/json"` — StartsWith — satisfies a request for `to:"json-ld"`), and as a side effect stashes the matched `*Format` onto the in-flight `maps.It` callback (`m.match = f`) so downstream code can see which format rule fired. `To(format, params)` / `From(format, params)` are constructor helpers used outside of tags (programmatic mapping calls).

## 7. Logs (`logs/`)

Declarative structured logging built on `github.com/go-logr/logr`, confirming the project `CLAUDE.md` claim that logging is `FilterStage`-ordered — exact constant is **`miruken.FilterStageLogging`** (`logs/filter.go:59`; the numeric value lives in `core-dispatch.md`'s filter-stage table, not redefined here).
- `logs.Factory` (`logs/factory.go`) — `NoConstructor()` marks it non-implicitly-constructable (must be explicitly registered by the Installer with the configured root `logr.Logger`); `NewContextLogger(p *provides.It) logr.Logger` is itself a **`provides.It` handler method** — i.e. `logr.Logger` is resolved through the same DI (`Provides`) pipeline as any other dependency, named by owner type (`p.Owner()`) when available.
- `logs.Emit` (`logs/filter.go`) is a `FilterProvider`: `AppliesTo` restricts it to `*handles.It` callbacks only (line 44); `InitWithTag` reads `logs:"verbosity=N"`. Its `filter.Next` (lines 62-101) resolves a `logr.Logger` via `provides.Type[logr.Logger](ctx)`, skips entirely if not found or if the verbosity level isn't `Enabled()` (cheap early-out before any string formatting), then logs "handling"/"completed"/"failed" with elapsed time (`miruken.Timespan`) around `next.Pipe()` — including the async branch (`promise.Then`/`promise.Catch` around `pout`, lines 91-99) so async handlers are timed correctly.
- `logs.Feature(rootLogger, ...config)` registers `Factory` as both a spec and a pre-built handler instance, plus registers `&Emit{verbosity: ...}` as a global filter (`b.Filters(...)`, `logs/feature.go:22`) — i.e. logging applies framework-wide once the feature is installed, not per-handler opt-in.

## 8. Config (`config/`)

Koanf-backed configuration, resolved through `Provides` (matches the project's "features built via declarative seams, resolved through DI" philosophy).
- `config.Provider` interface (`config/feature.go:11-13`): `Unmarshal(path string, flat bool, output any) error` — the port; `config/koanf/provider.go` is the koanf adapter (`config.P(k *koanf.Koanf) config.Provider`), confirming koanf as the backing library. `koanf.Merge`/`MergeStrict` extend koanf's map-merge with `ConvertSlices` (turns `map[string]any` with all-integer keys like `{"0":a,"1":b}` into a real `[]any` slice — a quirk of merging flattened config sources).
- `config.Factory` (`config/factory.go`) — also `NoConstructor()`; its single method `NewConfiguration(_ *struct{ args.Strict; Load }, p *provides.It) (any, error)` is a **`provides.It` handler bound to `Load`**, meaning: any `Provides` request carrying a `*config.Load` constraint (path + flat/nested flag, tag `path:"some.path,flat"`) is satisfied by unmarshalling that koanf path into a new instance of the requested type (`p.Key().(reflect.Type)`), reflectively allocated via `reflect.New`. Results are cached in an `atomic.Pointer[map[loadKey]any]` using copy-on-write (mutex only taken on a cache miss, lines 94-132) — same lock-free-read pattern the core `bindingMap` uses (see `core-dispatch.md`). If the unmarshalled value implements `Validate() error`, it's called and any error surfaces as the resolution error.
- `config.Load` constraint (`Required()==true`, `Implied()==false`) — since it's `Required`, a caller/handler **must** explicitly request a `Load`-constrained config value; it won't be implicitly injected.

## 9. Cascade & Effect (side-effect model)

`Effect` (`effect.go:19-21`, root package) is the general port: `Apply(HandleContext) (promise.Reflect, error)`. Handler methods can return *anything* as a trailing "extra output" and the framework will try to coerce it into an `Effect` in one of two ways:
1. It already implements `Effect`.
2. It has a compatible `Apply(...)` method discovered reflectively (`getEffectMethod`, `effect.go:190-265`) — looked up once per type and cached in a `sync.Mutex` + copy-on-write `atomic.Pointer[map[reflect.Type]effectBinding]` (same pattern as config's cache and the core policy `bindingMap`). The discovered method is compiled into a `funcCall` (see `core-dispatch.md`) so invocation afterward is a normal reflective call, not repeated method-shape inspection.

Built-in effects:
- **`CascadeEffect`** (`effect.go:23-29`, fluent builder `miruken.Cascade(callbacks...)`) — re-dispatches a list of callbacks through `Command`/`CommandAll` against `ctx.Composer` (or an explicitly supplied handler via `.WithHandler`), aggregating any resulting promises with `promise.All`. This is the primitive: "as a side effect of handling this request, also dispatch these other callbacks."
- **`cascade.Messages`** (`cascade/effects.go`) — same idea specialized for the `api` message envelope: `cascade.Post(msgs...)` / `cascade.Publish(msgs...)` cascade via `api.Post`/`api.Publish` instead of raw `Command`/`CommandAll` (see `delivery-http-json.md` for what those do). `cascade.Callbacks` is a type alias for `miruken.CascadeEffect` (line 12).
- **`effect.Group(ctx, effects...)`** (`effect/group.go`) — combinator that applies multiple effects and aggregates their promises, same 0/1/many `promise.All` shape used everywhere else in this codebase for "maybe async, maybe several" aggregation.
- **`effect.Async(ctx, anything)`** (`effect/async.go`) — wraps any coercible effect (via `effect.Ensure` → `miruken.MakeEffect(effect, true)`) so its `Apply` runs inside a fresh `promise.New`, forcing async execution and only resolving `struct{}{}` (fire-and-forget semantics — the underlying effect's own resolved value is discarded, only completion/error matters).

## 10. Constraint helpers subpackage (`constraint/`)

Distinct from the root `constraint.go` (which defines the `Constraint` interface, `Named`, `Metadata`, `Qualifier[T]`, `ConstraintProvider`, `constraintFilter` — see `core-dispatch.md`). This subpackage is pure convenience:
- `constraint/alias.go`: type aliases re-exporting root types under shorter names for consumers who import `constraint` instead of the root `miruken` package — `constraint.Named = miruken.Named`, `constraint.Metadata = miruken.Metadata`, `constraint.Provider = miruken.ConstraintProvider`.
- `constraint/helper.go`: `First[T miruken.Constraint](src miruken.ConstraintSource) (T, bool)` — linear scan helper to pull the first constraint of a given concrete type off any `ConstraintSource` (e.g. used by `config.Factory.NewConfiguration` to pull `*Load` off of `p` — a `*provides.It`, which implements `ConstraintSource` via embedding `CallbackBase`).

No behavior lives here beyond convenience/readability; nothing about the constraint *matching* algorithm (that's `constraintFilter.Next` in root `constraint.go`).
