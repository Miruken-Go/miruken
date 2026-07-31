# Core Dispatch Engine (root package `github.com/miruken-go/miruken`)

Everything in Miruken — DI, HTTP, JSON, validation, security, event sourcing —
is a `Handler` that processes a `Callback` through a `Filter` pipeline. This
doc covers the root package: the one mechanism the rest of the framework is
built on.

## 1. Overview

`Handler.Handle(callback, greedy, composer)` is the single dispatch entry
point in the whole framework (handler.go:11). A `Callback` (callback.go)
carries intent — "handle this command", "provide a value of this type",
"create an instance" — plus a `Policy` that defines *matching rules* between
the callback's key and the keys handler methods were bound under:

| Policy | Matching rule | Used by |
|---|---|---|
| Covariant (covar.go) | binding's output type assignable **to** the key | `provides` — DI resolution |
| Contravariant (contravar.go) | key assignable **to** binding's declared input type | `handles` — commands/events |
| Bivariant (bivar.go) | both input and output match, via `DiKey{In,Out}` | `maps` (transforms) |
| Invariant (invar.go) | exact key equality only | strict lookups |

A handler type's methods are introspected once via reflection at setup time
(describe.go) into an immutable `Binding` per method. Dispatch at request
time walks pre-built binding lists (policy.go) and runs a filter pipeline
(filter.go) — no struct-tag or type-shape parsing happens per call, only
`reflect.Value.Call` and argument-value resolution (see §7 and §12).

## 2. Core interfaces & types

- **`Handler`** (handler.go:11) — `Handle(callback any, greedy bool, composer Handler) HandleResult`. `composer` is the "ambient" handler used to resolve dependencies/services during the call (usually the root of a handler chain); passing a different composer than `this` lets middleware substitute a decorated view of the handler graph without changing `this`.
- **`ToHandler(any) Handler`** adapts a plain value with dispatchable methods into a `Handler` via `handlerAdapter` + `DispatchCallback` (handler.go:71-123) — this is how any handler-shaped struct passed to `setup.Specs(...)` becomes callable without explicitly implementing `Handler`.
- **`HandleContext`** (handler.go:19) — per-invocation info visible to filters/args: `Handler` (receiver instance), `Callback`, `Binding`, `Runtime *HandlerRuntime`, `Composer`, `Greedy`.
- **`Callback`** interface (callback.go:15) — `Key() any`, `Source() any`, `Target()/TargetForWrite()`, `Policy() Policy`, `Result(many bool) (any, *promise.Promise[any])`, `ReceiveResult(...)`, plus embeds `ConstraintSource` (constraint.go). `CallbackBase` (callback.go:52) is the common struct embedded by `Handles`, `Provides`, `Creates`, etc.
  - Results accumulate via `AddResult`/`ReceiveResult` under a `sync.Mutex` (callback.go:53), supporting async results (promises tracked in `c.promises`, `c.async atomic.Bool`) and multi-result "greedy" dispatch (slice/array squashing via `expandResults`, `processResults`).
  - `ensureResult` (callback.go:230) is the lazy result-materialization path: snapshots results under lock, releases the lock before calling `unwrapResult` (which can block on `AwaitAny`), then writes back with a double-checked lock — a deliberate avoid-deadlock-with-promise-callbacks pattern (see comment at callback.go:238-241).
  - `isSliceOrArray`/`processResults` fast-path `[]any` and `expandResults` before falling back to `reflect.ValueOf(s).Len()/Index(i)` for arbitrary typed slices (callback.go:348-375) — noted as a perf-relevant fast path in the source comments themselves.
- **`HandleResult`** (result.go) — a value type `{handled, stop bool; err error}` with monadic combinators: `.Or` (union — handled if either), `.And` (intersection — handled only if both), `.Then/.Otherwise/.OtherwiseIf/.OtherwiseHandledIf` for conditional chaining, `.WithError`. `stop` short-circuits further dispatch (used e.g. by `constraintFilter.Next → next.Abort()`). This is the framework's uniform "did anything handle this, and was there an error" signal — every dispatch call returns one.
- **`Disposable`** (dispose.go) — trivial `Dispose()` interface + `DisposableFunc`.
- **`Init`** (initialize.go) — marker type used as first-or-second arg to flag a method as an initializer (see §11 initializer). `initializer` Filter (order `FilterStageCreation-1100`) invokes `Constructor` then any `Init*` methods on the freshly created receiver, awaiting the first async one and then firing the rest inline off that promise's continuation.
- **`Key`** (key.go) — a `DependencyResolver` that resolves a value by string key (`ProvidesBuilder.WithKey(string(k))`); implements `InitWithTag` reading an `of:"..."` tag — i.e. `Key` itself is a constraint-style tag-driven type, usable as a dependency-arg annotation (`dep *struct{ miruken.Key `+"`"+`of:"foo"`+"`"+` }`).
- **`Predicate[T]`** (predicate.go) — generic `func(T) bool` + `CombinePredicates` (OR-combination).

## 3. The annotation-simulation binding system

Go has no annotations. Miruken simulates them with an **unused pointer-to-
anonymous-struct parameter**: the field list of that struct *is* the ordered
set of "annotations" on the method, and struct tags on those fields *are*
the annotation arguments.

```go
func (p *PassThroughRouter) Pass(
    _ *struct {
        handles.It
        miruken.SkipFilters
        Routes `scheme:"pass-through"`
    }, routed Routed, composer miruken.Handler,
) (any, miruken.HandleResult) { ... }          // api/route.go:106-121
```

The parameter is always `_` — it is **never allocated or dereferenced**.
Only its `reflect.Type` is inspected, and only once, at setup time. This
gives:

- **Compile-time checking** — embedded/tagged types must actually exist and
  be importable; a typo is a compile error, not a silently-ignored comment.
- **Zero runtime cost** — no allocation, no per-call reflection; the parsed
  result is baked into an immutable `Binding` (§4).
- **IDE navigability** — "go to definition" on `handles.It` or `Routes`
  works the way it never would on a string-based annotation.
- **Composability** — multiple "annotations" are just multiple struct
  fields; a reusable bundle of them is a **named** struct implementing
  `DefinesBindingGroup()` (a `BindingGroup` marker), analogous to a Java
  meta-annotation. Example: `es/command/handles.go`'s `Handles` type:
  ```go
  type Handles struct {
      miruken.BindingGroup
      handles.It
      Metadata
      processProvider
  }
  ```
  Any handler method can now just embed `command.Handles` instead of
  repeating `handles.It` + `Metadata` + `processProvider` every time.

### How parsing works (bind.go)

1. `bindingSpecFactory.createSpec(typ, minArgs)` (bind.go:240) looks at
   `typ.In(minArgs-1)` (the "spec" parameter position). It qualifies as a
   policy spec if it is a `Pointer` to a type that is either an **unnamed
   struct** (`at.Name() == ""`) or **implements `DefinesBindingGroup()`**
   (bind.go:249-252). A bare `*handles.It` (no wrapping struct) is instead
   handled by the "is it a Callback arg?" branch (bind.go:264-271), which is
   why `_ *handles.It, callback any` (no struct) also works — it's the
   degenerate case of "one annotation, no extra options".
2. `parseSpec` → `parseStruct` (bind.go:335-395) walks `typ.Fields()`, which
   Go's `reflect` **flattens embedded fields** through, including nested
   `BindingGroup` structs recursively (bind.go:365-377, tags accumulate via
   `internal.MergeStructTagsWith` as it recurses so an outer tag on a
   `BindingGroup` field still reaches inner fields).
3. For each field, a chain of `BindingParser`s is tried in order until one
   claims it (`bound = true`):
   - `bindingSpecFactory.parse` (bind.go:311) — field type is a `Callback`
     (e.g. `handles.It`) → `addPolicy(policy, field)`. A `key:"..."` tag on
     that field creates an additional **named policyKey** so the *same*
     method can register under multiple keys — see api/schedule.go:107-112
     where four separate `_ creates.It \`key:"..."\`` fields register one
     method as the constructor for four different types.
   - `parseFilters` (bind.go:397) — field implements `Filter` directly (rare;
     reads a `filter:"required,order=N"` tag) or `FilterProvider` → builds
     it via `internal.NewWithTag(type, mergedTag)`, i.e. `reflect.New` +
     call `InitWithTag(tag reflect.StructTag)` if implemented. This is where
     `Routes.InitWithTag` reads `scheme:"pass-through"` (api/route.go:57-65),
     `Named.InitWithTag` reads `name:"..."` (constraint.go:74-80), `Single`
     reads `mode:"covariant"` (lifestyle.go:99-104).
   - `parseConstraints` (bind.go:449) — field implements `Constraint` →
     same `NewWithTag` construction, `addConstraint`.
   - `parseOptions` (bind.go:474) — field is exactly `Strict`/`Optional`/
     `SkipFilters` → sets a `bindingFlags` bit, no allocation needed.
   - Anything left over becomes `Metadata` — `addMetadata` wraps the field's
     own type (if it implements nothing recognized) as arbitrary binding
     metadata (bind.go:518-533).
4. `bindingSpec.complete()` (bind.go:232) appends a `ConstraintProvider`
   collecting everything gathered into `b.constraints`, so multi-constraint
   matching runs as one filter (`constraintFilter`, order `FilterStage=0`,
   filter.go / constraint.go:167-231): it requires every constraint the
   *callback* declares to be satisfied by some constraint the *binding*
   declares (`Constraint.Satisfies`), and every **implied** binding
   constraint (`Implied() == true`, meaning "derivable from `HandleContext`
   alone, no explicit callback constraint needed") must independently hold.

`describe.go`'s `TypeSpec.newRuntime`/`FuncSpec.newRuntime` call
`factory.createSpec(methodType, 2)` (2 = skip receiver + spec param) for
every exported method, and `createSpec(ctorType, 2)` for `Constructor`. The
parsers list itself is assembled once in
`HandlerRuntimeFactoryBuilder.Build()` (describe.go:654-668): the
`bindingSpecFactory` itself, then `parseOptions`, `parseFilters`,
`parseConstraints`, then any user-supplied extra `BindingParser`s — so the
annotation vocabulary is *extensible* by installing custom parsers through
that builder.

Two more real examples:
- `security/authorizes/it.go` defines its own `It`/`Builder`/`Options` (its
  own bespoke Callback+Policy, contravariant) rather than reusing
  `handles.It` — worth knowing this pattern (custom Callback+Policy pair) is
  also idiomatic, not just the three built-ins.
- `es/command/handles.go`'s `Handles` `BindingGroup` bundles a `processor`
  filter (order 10) that runs after the handler method returns and prints
  the output — showing a `BindingGroup` can carry actual behavior (a
  `FilterProvider`), not just markers.

## 4. describe.go — HandlerRuntime construction

`HandlerRuntime` (describe.go:16) is the per-handler-type (or per-function)
compiled metadata: a `policyBindingMap` (`bindings`) plus a `filterBindingGroup`
(`compound`, for dynamic `Filter` methods discovered on the handler itself —
see §8).

- **`TypeSpec`** (describe.go:40) drives `newRuntime` for struct/pointer
  handler types:
  1. If a `Constructor` method exists, its spec is parsed with `createSpec(ctorType, 2)`.
  2. Unless a `NoConstructor` method exists, a bare `providesPolicyIns`
     policyKey is added implicitly — **every constructable type is
     auto-registered as providable by convention**, unless explicitly opted out.
  3. Every other exported method is checked with `createSpec(methodType, 2)`.
     - Methods with no valid spec but named `Init*...` (or whose 2nd arg is
       `Init`) are collected as `inits` (project convention: "`Init*`
       methods are implicit initializers", CLAUDE.md).
     - Methods with no valid spec that aren't handler methods, and the type
       doesn't itself implement `Filter`, are checked by `parseFilterMethod`
       for the *dynamic Filter method* convention (§8) and folded into
       `runtime.compound`.
     - Methods **with** a valid spec register one `Binding` per `policyKey`
       via the policy's `MethodBinder.NewMethodBinding`.
  4. Constructor bindings are built **after** methods (so `inits` is fully
     collected) via `ConstructorBinder.NewCtorBinding`.
- **`FuncSpec`** (describe.go:196) does the analogous thing for a bare
  function passed to setup (first arg is the spec parameter, `createSpec(funType, 1)`).
- **`mutableHandlerFactory`** (describe.go:502) memoizes `HandlerRuntime` by
  `spec.key()` (`reflect.Type` for types, `fun.Pointer()` for functions) —
  introspection happens exactly once per distinct handler type/func for the
  lifetime of the factory. `HandlerRuntimeFactoryBuilder` also supports
  `Observers(...)` for `HandlerRuntimeCreated/Binding/Registered` hooks
  (used by things like OpenAPI generation to walk all discovered bindings —
  see delivery doc).
- Handler-runtime dispatch itself (`HandlerRuntime.Dispatch`, describe.go:279)
  is where the per-callback binding walk, constraint/guard checks, filter
  ordering (`orderFilters`), and `binding.Invoke`/`pipelineInvoke` happen —
  this is the request-time hot path; see §12.

## 5. policy.go — binding storage & lookup

- **`indexedBindingList`** (policy.go:43) is a hybrid structure per `Policy`:
  - `variant list.List` — a doubly linked list of bindings whose key is a
    `reflect.Type` (partially ordered by `Policy.Less`, most-specific
    generally toward the front — see `insert`, policy.go:73).
  - `invariant map[any][]Binding` — bindings keyed by exact non-type keys
    (e.g. string keys from `Key`).
  - `index map[any]*list.Element` — direct-key → list-node shortcut for
    types actually used as a binding key (registered at insert time).
  - `dynIndex atomic.Pointer[map[any]*list.Element]` + `dynLock sync.Mutex`
    — **copy-on-write** cache of *interface-relationship* lookups: if a
    callback key is an interface a binding's concrete key satisfies, that
    can only be discovered by a *linear scan* the first time; the result is
    then memoized into `dynIndex` (policy.go:135-158) so subsequent lookups
    for that exact key are O(1). Comment: "reads should be more frequent
    than writes."
- **`reduce`** (policy.go:110) is the actual matching walk used by
  `HandlerRuntime.Dispatch`: try the direct `index`, then `dynIndex`, else
  fall back to scanning from `variant.Front()` and memoize. Then check
  `invariant[key]`. Then always also check bindings registered under the
  wildcard `internal.AnyType` key (any-typed catch-alls).
- **`policyBindingMap`** (policy.go:53) is just `map[Policy]*indexedBindingList`,
  one per policy instance (there's one process-wide `handlesPolicyIns`,
  `providesPolicyIns`, `createsPolicyIns`, etc. — see `var ... Policy = &...{}`
  at the bottom of handles.go/provides.go/creates.go).
- **`DispatchPolicy`** (policy.go:224) is the generic "given a handler
  instance and a Callback, find its HandlerRuntime via
  `CurrentHandlerRuntimeFactory(composer)` and dispatch" — this is what
  every built-in Callback's `Dispatch` method ultimately calls unless the
  handler implements `PolicyDispatch` itself (e.g. `inferenceHandler`, §6).
- **`acceptResultsWithEffects`** (policy.go:242) is the shared result-
  reduction logic used by Contravariant/Invariant/Bivariant policies'
  `AcceptResults`: interprets a method's returned `(value, HandleResult)`,
  `(value, error)`, or plain value, and separates out `Effect`s (§11) from
  the primary result.

## 6. Variance policies — exact matching rules

All four live directly on `Policy` implementations; `MatchesKey(key,
otherKey, invariant) (matches, exact bool)` is the core predicate.

- **`CovariantPolicy`** (covar.go) — `MatchesKey`: exact match, or (if not
  strictly invariant) `key.AssignableTo(otherKey)` — the **candidate's
  declared type is assignable to the requested key type**, i.e. "can this
  binding's output satisfy a request for T" reads as "T's key is assignable
  from the binding's key" in the actual code it's `bt.AssignableTo(kt)`
  where `bt` = *binding* key, `kt` = the *lookup* key — so a binding
  providing `*Foo` matches a lookup key `Fooer` interface. `VariantKey`
  treats `any` (interface{}) as the special "unknown/wildcard" key
  (`internal.IsAny`). Used by `provides`/`creates`.
- **`ContravariantPolicy`** (contravar.go) — inverse direction:
  `kt.AssignableTo(bt)` where `kt` is the incoming callback key and `bt` is
  the binding's declared input type — "this handler declared it accepts
  type X; does the incoming callback's concrete type satisfy X" — also
  allows a pointer-vs-value elem match (`kt.Kind()==Pointer &&
  kt.Elem().AssignableTo(bt)`). `Strict() == true` (contravar.go:52) — no
  implicit result-slice-expansion (differs from Covariant, whose `Strict()
  == false`). Used by `handles`.
- **`BivariantPolicy`** (bivar.go) — key is a `DiKey{In, Out}` pair; matches
  only if **both** the `In` half satisfies a `ContravariantPolicy.MatchesKey`
  and the `Out` half satisfies a `CovariantPolicy.MatchesKey` — it literally
  embeds one of each (`in ContravariantPolicy; out CovariantPolicy`,
  bivar.go:20-24) and delegates. Used by `maps` (type-to-type transforms:
  the input type and desired output type must both line up).
- **`InvariantPolicy`** (invar.go) — `MatchesKey` only ever returns `key ==
  otherKey`; no assignability check at all, no `Less` ordering (`Less`
  always `false`). Used for strict, non-polymorphic lookups.

`VariantKey(key) (variant, unknown bool)` tells `indexedBindingList.insert`
whether a key belongs in the `variant` list (type-based, orderable) or the
`invariant` map (string/other exact keys), and whether it's the special
"unknown"/wildcard key (`any`) that should go at the tail of the list rather
than being position-sorted.

## 7. funcCall & reflection invocation

This is the layer that actually calls handler code via reflection. **Split
clearly into bind-time (once) vs dispatch-time (every call) work.**

### Bind-time (once per handler method/function/constructor)

- `newFuncCall(fun reflect.Value, args []arg)` (funcbind.go:44-52) caches
  `fun.Type()` and **every** `funType.In(i)` into `argTypes []reflect.Type`
  — so no `.Type()`/`.In(i)` calls happen again at dispatch time.
- `args []arg` is built once by `buildDependencies` (arg.go:294-347), which
  for each remaining parameter of the method:
  - detects `promise.Inspect(argType)` (is this parameter `*promise.Promise[X]`?) and
    sets `bindingAsync` flag + `spec.logicalType` once,
  - detects a dependency-spec struct pointer (`*struct{ ... }` matching the
    same anonymous-struct convention as §3, but for **dependency
    arguments** — parsed via `dependencyParsers = {parseOptions,
    parseResolver, parseConstraints}`, arg.go:288-292) so a dependency
    parameter can itself carry `Optional`/`Strict`/a custom
    `DependencyResolver`/`Constraint`s,
  - resolves to a concrete `arg` implementation: `zeroArg` (placeholder,
    e.g. the spec param itself), `CallbackArg` (raw callback), `sourceArg`
    (callback's `Source()`), or `DependencyArg` (general DI resolution via
    `defaultDependencyResolver` unless a custom `DependencyResolver` was
    attached).
  - `validateCovariantFunc`/`validateContravariantFunc`/`validateBivariantFunc`/
    `validateInvariantFunc` (covar.go:162, contravar.go:105, bivar.go:114,
    invar.go:87) each run once at bind time to determine the binding `key`
    (from return type, unless already fixed by a `key:"..."` tag) and to
    validate output-type shape (single value + optional error/HandleResult
    trailer; effect outputs recognized via `ValidEffect`).
- Result: a `MethodBinding`/`FuncBinding`/`CtorBinding` (methodbind.go,
  funcbind.go, ctorbind.go) holding the cached `funcCall` and `BindingBase`
  (filters/constraints/flags already resolved). **`Binding` objects are
  immutable and shared** across every dispatch to that method.

### Dispatch-time (every call) — `funcCall.Invoke` (funcbind.go:84-97)

1. `resolveArgs(len(initArgs), ctx)` (funcbind.go:99-141) — for **every**
   arg, calls `arg.resolve(argType, ctx)`. For `DependencyArg` this walks
   into `defaultDependencyResolver.Resolve` (arg.go:218-250), which builds a
   **new `ProvidesBuilder`** and issues a nested `Provides` callback
   (`p.Resolve(ctx, many)`) — i.e. **resolving one dependency argument is a
   full recursive `Handler.Handle` dispatch**, including its own binding
   lookup, filter pipeline, etc. This is the single biggest source of
   repeated reflection/dispatch overhead: an N-argument handler method
   triggers up to N nested dispatches every single invocation.
   - Each resolved value is a fresh `reflect.Value` (`reflect.ValueOf(result)`,
     arg.go:264 / defaultResolver) — allocation-per-arg-per-call is intrinsic
     to the current design.
   - Async args get lifted/coerced via `promise.Lift`/`promise.CoerceType`
     (funcbind.go:113-124) even in the common synchronous case if the
     `bindingAsync` flag is set — always constructs a promise wrapper.
2. `callFuncWithArgs(fun, ra, initArgs)` (funcbind.go:146-168) — allocates a
   fresh `[]reflect.Value` sized `len(initArgs)+len(ra)`, converts every
   `initArgs[i]` via `reflect.ValueOf(ia)` (**even values that are already
   `reflect.Value`-wrapped upstream get re-boxed** — no fast path for
   already-typed values), calls `fun.Call(in)` (the actual reflection call,
   unavoidable in this design), then unboxes every result via
   `v.Interface()` into a fresh `[]any`.
3. `mergeOutput`/`mergeOutputAwait` (funcbind.go:181-281) normalize
   `(out, promise, err)` triples — checks last-output-is-error and
   first-output-is-promise via type assertions on every call.

`MethodBinding.Invoke` (methodbind.go:53-64) additionally does an
`initArgs = append(initArgs, nil); copy(initArgs[1:], initArgs);
initArgs[0] = ctx.Handler` shuffle **every call** to prepend the receiver —
a small extra allocation/copy per dispatch when `initArgs` is non-empty.

`CtorBinding.Invoke` (ctorbind.go:48-69) calls `reflect.New(typ.Elem())` (or
`reflect.New(typ).Elem()`) **every time** a new instance is requested — this
is the actual "new instance" allocation path for transient-lifestyle types,
guarded only by a same-type receiver check to avoid double-construction
during initializer chains.

`infer.go`'s `inferenceHandler`/`inferenceGuard`/`methodIntercept` is a
separate dispatch path used when a callback isn't matched by any directly-
registered binding: it wraps every discovered binding across *all*
registered handler specs into one synthetic `HandlerRuntime` so an
unrecognized-but-plausible callback can still be inferred against the whole
universe of known handler types (guarded against duplicate visits per
handler type via `inferenceGuard.resolved map[reflect.Type]struct{}`).

`MakeCaller` (call.go) exposes the same `funcCall` machinery as a public
`CallerFunc` for calling an arbitrary function with dependency injection
outside the normal binding/policy path — used e.g. for one-off invocation
of user-supplied functions (setup bootstrapping, some filter constructors).

## 8. Filter pipeline (filter.go)

- **`Filter.Next(self, next Next, ctx, provider) ([]any, *promise.Promise[[]any], error)`**
  — every filter gets a `next Next` continuation function
  (`next(composer, proceed, values...)`), the current `HandleContext`, and
  its owning `FilterProvider`. `self` is passed for late-bound dynamic
  dispatch (see `FilterAdapter`, below).
- **Stage constants** (filter.go:17-23): `FilterStage=0`,
  `FilterStageLogging=1000`, `FilterStageAuthorization=3000`,
  `FilterStageValidation=5000`, `FilterStageCreation=math.MaxInt32` — these
  match CLAUDE.md's documented ordering; `FilterStageCreation` intentionally
  sorts dead last (constructors/initializers/lifestyle caching run *around*
  the actual handler body, e.g. `Lifestyle.Order()` is
  `FilterStageCreation-10000`, `initializer.Order()` is
  `FilterStageCreation-1100` — both still logically "at the edge" but the
  lifestyle filter wraps *outside* the initializer).
- **`Next.Pipe`/`.PipeAwait`/`.PipeComposer(Await)`** — convenience wrappers
  calling `next(nil/composer, true, values...)` and normalizing via
  `mergeOutput`/`mergeOutputAwait`. **`Next.Abort()`** calls `next(nil,
  false)`, which every pipeline's closure turns into a `RejectedError` —
  this is how `constraintFilter` and `Routes` reject non-matching callbacks
  without it being treated as an application error.
- **`orderFilters`** (filter.go:297-388) merges filters from every
  applicable source — binding-level, handler-runtime-level (`h.filters`),
  policy-level, plus a synthetic compound-handler/self-as-Filter provider —
  applies `SkipFilters` short-circuiting (either from `FilterOptions` set
  via `DisableFilters`/`EnableFilters`/`ProvideFilters`, or the binding's own
  `SkipFilters()` flag), filters out non-`Required()` providers whose
  `AppliesTo(callback)` returns false, then **sorts by `Filter.Order()`**
  (ties broken toward the earlier-declared one; negative order sorts last —
  filter.go:370-386 quirk to note).
- **`pipelineInvoke`** (filter.go:390-420) builds the actual `next` closure
  chain by index, terminating in `binding.Invoke(ctx)` — this closure chain
  is built **fresh on every dispatch** (not cached across calls), since the
  filter set can vary per-call (composer-supplied `FilterOptions`).
- **Dynamic Filter methods** — a type can implement filtering behavior via
  ordinary methods rather than implementing `Filter.Next` directly.
  `parseFilterMethod` (filter.go:629-681) recognizes a method whose first
  (or second, if the first is a callback-source type) parameter is `Next`,
  returning `([]any, *promise.Promise[[]any], error)`, optionally also
  taking a `HandleContext` and/or `FilterProvider` parameter. `getFilterBinding`
  (filter.go:583-621) discovers **all** such methods on a type (excluding
  literally named `"Next"`, reserved) once, caches them by
  `reflect.Type` in a copy-on-write `atomic.Pointer[map[reflect.Type]filterBindingGroup]`
  (`filterBindingMap`), and notes: **"Methods in Go are sorted in
  lexicographic order which will determine the order of filter execution"**
  — i.e. if a dynamic-filter type has multiple such methods, their firing
  order is controlled by **method name alphabetical order**, not
  declaration order — a non-obvious gotcha worth remembering when adding
  methods to an existing dynamic filter type.
- **`compoundHandler`** — when a handler type itself has extra
  non-handler-method "impure" methods (matched via `parseFilterMethod` in
  `describe.go`'s method loop, stored in `runtime.compound`), those are
  wrapped as a synthetic filter around the *whole* handler dispatch (see
  §4/HandlerRuntime.Dispatch, filter.go:330-343) — this is the "compound
  handler" mentioned in describe.go, splitting "pure" decision logic from
  "impure" I/O side effects for testability (filter.go:467-469 doc comment).
- **`FilterOptions`** (filter.go:270) + `UseFilters`/`ProvideFilters`/
  `DisableFilters`/`EnableFilters` are the `Options(...)`-pattern (see §11)
  builders consumers use to inject ad hoc filters or globally disable
  non-required ones for a handler subtree.

## 9. Lifestyles (lifestyle.go, provides.go)

- **Transient** (default) — no lifestyle filter attached; `CtorBinding.Invoke`
  allocates a fresh instance every resolution.
- **`Single`** (lifestyle.go:67) — a `LifestyleProvider` (itself a
  `FilterProvider` that only `AppliesTo` `*Provides` callbacks). Reads a
  `mode:"covariant"` tag via `InitWithTag`. `InitLifestyle(binding)`
  (`LifestyleInit` interface) decides between:
  - `single` — one `sync.Once`-guarded `singleEntry` — used when the
    binding's key is a concrete type.
  - `singleCovar` — a **copy-on-write `singleCache map[any]*singleEntry`**
    keyed by the *requested* key (since a covariant provider can be resolved
    through many different interface/base-type keys, e.g. resolving `Fooer`
    vs `*Foo` vs `any` all reaching the same singleton) — used automatically
    when the binding's key is `any` (`internal.IsAny`), or forced via the
    `mode:"covariant"` tag. New key lookups first scan existing cached
    instances for assignability before creating a brand-new singleton
    branch (lifestyle.go:159-172), so `Resolve[Fooer]()` and
    `Resolve[*Foo]()` on the same provider share one instance once either
    has been created.
  - `singleEntry.get` (lifestyle.go:191-215) — `sync.Once.Do` wraps the
    inner `next.Pipe()` call; on panic or empty result it **replaces the
    `sync.Once`** so a failed construction can be retried on the next
    resolution rather than being poisoned forever.
- **`providesPolicy.NewCtorBinding`** (provides.go:287-304) — if the handler
  type has **no explicit binding spec at all** (`spec == nil`, i.e. it was
  purely auto-registered by the implicit-Provides convention, §4 step 2),
  it force-attaches `&Single{}` — meaning **every implicitly-registered
  constructable type defaults to singleton lifestyle** unless the type
  explicitly declares a `Provides` spec (which would then need to opt into
  `Single` itself via an embedded field). This is a load-bearing default
  worth calling out explicitly since it's easy to assume "transient by
  default" from other DI frameworks.
- **Scoped lifestyle** lives in the `context/` package (see
  setup-context.md) — it hooks in the same way via `LifestyleInit` but ties
  instance lifetime to a `context.Context` node rather than process lifetime
  or a fixed cache.

## 10. Handles / Provides / Creates

Three built-in `Callback`+`Policy` pairs, each with: a struct
(`Handles`/`Provides`/`Creates`), a `*Builder`, package aliases in
`handles`/`provides`/`creates` (thin re-export packages — `It = miruken.X`,
plus free functions delegating to the root package, e.g.
`handles.Command` → `miruken.Command`).

- **`Handles`** (handles.go) — Contravariant. `Command`/`CommandAll` (no
  result expected) and `Execute[T]`/`ExecuteAll[T]` (typed result via
  `IntoTarget`) build a `Handles` callback and call `handler.Handle`.
  `CanDispatch`/`CanInfer`/`CanFilter`/`CanBatch` all delegate to the
  *wrapped* `callback any` if it implements the matching optional
  interface — letting an inner domain callback opt out of inference/
  filtering/batching transparently through the `Handles` envelope.
- **`Provides`** (provides.go) — Covariant, with extra DI-specific state:
  `parent *Provides` (for nested dependency chains — `Trigger()` walks up
  to find the originating callback), `owner any` (the handler instance that
  triggered this resolution, used by dependency resolvers), `explicit bool`
  (see `Explicit` sentinel — suppresses the "auto-match by-type if handler
  itself is assignable" shortcut in `Dispatch`, provides.go:110-119, forcing
  full binding-based resolution even for trivial cases). `Resolve[T]`/
  `ResolveKey[T]`/`ResolveAll[T]` (free functions) are the primary DI entry
  points used throughout the framework (e.g. by `DependencyArg` itself).
- **`Creates`** (creates.go) — Covariant, simpler than `Provides` (no
  parent/owner/lifestyle concerns — creation is meant to be a pure "give me
  a fresh instance", typically for polymorphic deserialization targets or
  explicit factories). `Create[T]`/`CreateKey[T]`/`CreateAll[T]`.
- **`Explicit`** (provides.go:349, a `var Explicit explicit` sentinel value)
  — passed as a constraint to force non-implicit resolution.

## 11. Supporting root-package files (brief)

- **axis.go** — `TraversingAxis`-scoped `Handler` decoration (`Axis(...)`,
  `SelfAxis`/`RootAxis`/`ChildAxis`/etc. builders) for handler-tree
  traversal-aware dispatch; `Publish = ComposeBuilders(SelfOrDescendantAxis,
  Notify)` is the standard "broadcast to self and descendants" builder.
- **batch.go** — `Batch`/`BatchAsync`/`BatchTag(Async)` implement request
  coalescing: a `batchHandler` intercepts a `*batch` `Provides` lookup or
  any type implementing `batching` (`CompleteBatch(Handler)`), accumulating
  work via `GetBatch[TB]` until `Complete()` fires all `CompleteBatch` calls
  concurrently via `promise.All`. `Batched[T]` wraps a per-item callback
  participating in a batch (used e.g. by `api/route.go`'s `batchRouter`).
  `NoBatch` builder opts a subtree out.
- **compose.go** — `Composition`/`CompositionScope`: wraps a callback so
  handlers can recognize "this callback arrived via the ambient composer,
  not directly" — used to avoid infinite handler recursion and to let
  cross-cutting handlers (semantics, options, batch, filter) distinguish
  their own re-dispatch from a fresh external call.
- **constraint.go** — see §3; also defines `Named` (name-tag match),
  `Metadata` (arbitrary key/value tag match via `metadata:"k=v,k2=v2"`),
  `Qualifier[T]` (type-based marker constraint with no runtime state).
- **ctorbind.go** — see §7.
- **effect.go** — `Effect` is a secondary output channel for handler
  results representing side-effecting intents (e.g. "cascade this callback
  to other handlers" via `CascadeEffect`/`Cascade(...)`, used by
  `cascade/` package). Like filters, a type can implement `Effect.Apply`
  directly or provide a **dynamic `Apply` method** discovered via
  reflection once and cached in a copy-on-write `atomic.Pointer` map
  (`effectBindingMap`) — mirrors the dynamic-Filter-method pattern in §8.
  `MakeEffect`/`MakeEffects`/`ValidEffect` are how a plain returned value
  gets recognized/promoted into an `Effect` during `AcceptResults`.
- **graph.go** — generic tree traversal (`Traversing` interface:
  `Parent()`/`Children()`/`Traverse(axis) iter.Seq[Traversing]`),
  independent of Handler/Callback; implements pre-order/post-order/
  level-order/reverse-level-order and the axis-relative traversals
  (siblings, ancestors, descendants) that back `axis.go` and `context/`
  tree traversal. **REDESIGNED (2026-07-31, Simplify Pass, BREAKING root
  API change):** `TraverseAxis`/`TraversePreOrder`/`TraversePostOrder`/
  `TraverseLevelOrder`/`TraverseReverseLevelOrder` now return
  `iter.Seq[Traversing]` instead of taking a `TraversalVisitor` and
  returning `error` — callers write a plain `for node := range
  TraversePreOrder(root) { ... break ... }` instead of implementing
  `TraversalVisitor`/wrapping a func in `TraversalVisitorFunc` (both
  removed). Circularity detection (`traversalHistory`,
  `TraversalCircularityError`) now **panics** instead of returning an
  error — confirmed via repo-wide grep that nothing anywhere caught it
  specially, so this trades one unhandled-error class for one
  unhandled-panic class with the same observable outcome, consistent
  with the panic-for-invariant-violation idiom already used in
  `security/authorizes/filter.go`'s async path. `context.Context.HandleAxis`
  is the one place that publicly promised "returns an error, never
  panics"; it now has a targeted `recover()` that converts a
  `TraversalCircularityError` panic back into the same
  `HandleResult.WithError(...)` it produced before — its own external
  behavior is unchanged for every caller. `context.Context.Traverse`'s
  signature changed to match. Verified zero usage in the local
  demo-consumer repos (`adb2c`/`team`/`team-srv`), so no known real-world
  breakage. One deliberate, minor behavior refinement as a side effect:
  the old implementation was inconsistent about honoring a visitor's
  `stop=true` across different traversal orders (some ignored it,
  untested either way since every existing visitor always returned
  `stop=false`) — the new `yield`-based design makes `break` stop
  consistently and correctly at every level, a small correctness fix,
  not an intentionally-preserved quirk.
- **methodbind.go** — see §7.
- **options.go** — the `Options(...)`/`GetOptions[T]`/`GetOptionsInto`
  pattern: a struct-typed "settings" `Builder` is installed via
  `BuildUp(handler, Options(MyOpts{...}))`, and retrieved anywhere downstream
  via `GetOptions[MyOpts](handler)`, which issues a special `optCallback`
  that `optionsHandler.Handle` intercepts and merges into (via `mergo.Merge`
  + `optionMerger` transformer honoring a custom `mergeable.MergeFrom`).
  This is the mechanism behind `FilterOptions`, `authorizes.Options`, etc.
  — the generic "typed configuration you can push down the handler chain
  and read back anywhere" primitive.
- **resolves.go** — `Resolves` extends `Provides` to *also* dispatch the
  original triggering callback onto whatever gets resolved (used by
  `methodIntercept` in infer.go — resolving a handler by type and then
  actually invoking the callback on it, for the inference path).
- **semantics.go** — `CallbackSemantics`/`Broadcast`/`BestEffort`/`Notify`
  builders: `Broadcast` forces `greedy=true` regardless of caller-specified
  greediness; `BestEffort` swallows `NotHandledError`/`RejectedError` into a
  successful `Handled` result. `Notify = Broadcast | BestEffort` is the
  fire-and-forget "publish" semantic (paired with `Publish` in axis.go).
- **timespan.go** — trivial `Timespan` (typed `time.Duration`) with a
  `Format(layout)` helper using the Unix-epoch trick for duration formatting.
- **trampoline.go** — `Trampoline` is the common base for "callback that
  just wraps another callback and forwards everything" types (`Composition`,
  `noBatch`) — implements `Source/Policy/Result/SetResult/CanInfer/CanFilter/
  CanBatch/CanDispatch/Dispatch` all by delegating to the wrapped `callback any`.

## 12. Reflection & performance notes

Facts only — no prescriptions; a dedicated optimization pass is a separate
future task.

**Done once, cached, and reused for the process lifetime:**
- `reflect.Type` introspection of a handler type's methods (`describe.go`,
  keyed by `reflect.Type`/`fun.Pointer()` in `mutableHandlerFactory.handlers`).
- Struct-tag parsing for the "annotation" system (§3) — parsed into
  `bindingSpec` → baked into immutable `Binding`/`FilterProvider`/
  `Constraint` instances.
- `funcCall.argTypes` (funcbind.go:44-52) — every `funType.In(i)` cached.
- Dynamic-Filter-method discovery (`filterBindingMap`, filter.go) and
  dynamic-Effect-method discovery (`effectBindingMap`, effect.go) — both
  copy-on-write `atomic.Pointer[map[reflect.Type]...]`, populated lazily on
  first use of a given concrete type, then reused.
- `indexedBindingList.dynIndex` (policy.go) — interface-satisfaction lookups
  are memoized per discovered key after the first linear scan.

**Happens on every dispatch (per callback, and recursively per dependency argument):**
- `arg.resolve` for every parameter, including a **full nested
  `Provides`/`Handle` dispatch per dependency argument**
  (`defaultDependencyResolver.Resolve`, arg.go:218) — an N-argument handler
  method costs up to N recursive dispatch cycles, each with its own binding
  lookup and filter-pipeline construction.
- `callFuncWithArgs` (funcbind.go:146) — fresh `[]reflect.Value` allocation
  sized to argument count, `reflect.ValueOf` boxing of every `initArgs`
  element (including the receiver, and including values that may already
  have been `reflect.Value` upstream), the actual `fun.Call(in)`, then
  `v.Interface()` unboxing of every return value into a fresh `[]any`.
- `MethodBinding.Invoke`'s initArgs shuffle (methodbind.go:57-63) — an
  `append`+`copy` per call when init args are present, purely to prepend
  the receiver.
- `orderFilters` (filter.go:297) rebuilds and sorts the effective filter
  list **every dispatch** (not cached across calls) — necessary because
  `FilterOptions` can vary per-call via the composer, but means
  `slices.SortFunc` runs per dispatch even when the filter set is
  static in practice for a given binding+composer combination.
- `HandlerRuntime.Dispatch`'s `pb.reduce` (policy.go) walk plus
  `policy.MatchesKey` calls per candidate binding in the variant list
  before a match is confirmed (mitigated by the index/dynIndex, but a
  cold/unindexed lookup is still a linear scan).
- `CtorBinding.Invoke`'s `reflect.New` per transient resolution (expected —
  this is the actual object-allocation site — but worth naming explicitly
  since `Single`/singleton bypasses it after the first call).
- `isSliceOrArray`/`processResults` reflection fallback
  (`reflect.ValueOf(s).Len()/Index(i)`) for any handler-returned slice type
  other than `[]any`/`expandResults` — every element boxed via `.Interface()`.

**Candidate areas worth investigating for performance** (observations, not
recommendations):
- Dependency-argument resolution recurses through the full `Handle` →
  binding-lookup → filter-pipeline machinery even for the extremely common
  case of resolving `Handler`/`HandleContext`/the callback itself, which
  `DependencyArg.resolve` already special-cases *before* falling into the
  resolver (arg.go:174-196) — but anything beyond those already goes
  through a full nested dispatch.
- **INVESTIGATED AND DECLINED (2026-07-31, Performance Pass #1):** `orderFilters`
  recomputing and sorting per dispatch, even when the binding/handler-runtime/
  policy provider lists are static — looked like a caching win but is **not**
  safe to cache blindly: `filterSpecProvider.Filters` (filter.go:172-198) does a
  full DI `Provides.Resolve(composer, false)` on every call specifically so
  `Scoped`-lifestyle filters resolve against the *correct* ambient context;
  caching per binding would pin a scoped filter to whichever
  context/composer resolved it first. Several `AppliesTo(callback)` checks
  also key off the live callback. This flows through the security/validation
  filter pipeline, so left untouched rather than shipping an unverified
  change here — would need a narrower design (e.g. only cache when no
  `Scoped` filter and no data-dependent `AppliesTo` is present) before
  revisiting.
- `callFuncWithArgs`'s per-call `reflect.ValueOf` boxing of `initArgs`
  (which are frequently the same small set of types — e.g. always the
  receiver, always `ctx`) has no fast-path avoiding the boxing when the
  value's concrete type is already known at bind time.
- **FIXED (2026-07-31, Performance Pass #1):** the `MethodBinding.Invoke`
  initArgs `append`+`copy` shuffle (was methodbind.go:57-63) ran an `append`
  (which reallocates once, since these slices are typically exact-length
  literals) followed by a full self-copy shift, on every method-binding
  invocation with init args. Replaced with a single `make`+forward-`copy` —
  same one allocation, half the copy work, identical output. Zero behavior
  change; full test suite (all 3 modules) unaffected.
- No apparent memoization of `arg.flags()` combinations that are checked per
  resolved argument (`funcbind.go:113,119`) — cheap today (bitwise AND) but
  called in a hot loop per arg per call.
- **FIXED (2026-07-31, Performance Pass #1):** `promise.Then`/`Catch` (both the
  free functions in promise.go and the `Reflect`-interface instance methods
  in reflect.go) always spawned a goroutine + allocated a channel via `New`,
  even when the source promise was already permanently settled (`p.ch ==
  nil` — true for anything built via `Resolve`/`Reject`, and critically for
  every value from `promise.Lift`, which is exactly what `funcbind.go`'s
  `resolveArgs` produces when a dependency argument is typed
  `*promise.Promise[X]` but the value was available synchronously). Added an
  unexported `newSync` helper that runs the executor on the calling goroutine
  instead of spawning one, gated on `p.ch == nil`; the async path is
  byte-for-byte unchanged. Verified via a dedicated correctness sub-agent
  review (no unguarded data races — `p.ch == nil` is a genuine write-once-
  before-publish invariant; no code anywhere relies on `Then`/`Catch`
  results being *not yet* settled immediately after the call; `All`/`Race`
  behavior provably unaffected; panic-to-rejection semantics identical via
  the same `handlePanic`) and by benchmark
  (`promise/test/promise_bench_test.go`): `Then`/`Catch` on an already-
  resolved source dropped from ~800-880ns/11 allocs to ~220-230ns/9 allocs
  (~3.5x faster, 2 fewer allocations); the genuinely-async case is
  unchanged (~1830ns/21-22 allocs), confirming the fast path only fires
  where intended. `Coerce`/`CoerceType`/`Unwrap` were deliberately left out
  of scope (the `Reflect` interface has no way to expose "is `ch` nil"
  without a bigger interface change, and `Unwrap` doesn't call `Then`/`Catch`
  at all).

## Open questions / things not fully verified

- `filter.go`'s `filterBinding.invoke` loop at lines 501-507
  (`for i := len(initArgs); i <= len(initArgs)+1; i++`) has slightly unusual
  bounds — confirmed it's inserting `ctx`/`provider` at their recorded
  indices *after* the base `initArgs`, but didn't trace every edge case
  (e.g. both ctxIdx and prvIdx unset simultaneously with applyTo present).
- Did not trace `internal/` package internals (`internal.CopySliceIndirect`,
  `internal.CopyIndirect`, `internal.Exported`, `internal.TargetValue`,
  `internal/seq`, `internal/slices`) — these are referenced throughout but
  belong conceptually to a "runtime/reflection helpers" layer; worth a
  dedicated pass if deep performance work targets `internal/runtime.go`.
- `setup/`, `context/`, and how `Scoped` lifestyle integrates with
  `context.Context` are intentionally out of scope here — see
  `setup-context.md`.
