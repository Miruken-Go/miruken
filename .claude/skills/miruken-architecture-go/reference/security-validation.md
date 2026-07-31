# Security & Validation

Both packages are **Features** built declaratively on the core Handler/Callback/Filter/Policy
engine (see `core-dispatch.md`) and on the "policy-marker struct" annotation-simulation system
(anonymous `*struct{ handles.It; SomeConstraint \`tag:"..."\` }` first parameters, parsed once at
setup time by `bind.go`/`describe.go`). This file only shows how these two packages *use* that
machinery, not how it works internally.

## 1. Overview

- `security/` — a JAAS-flavored authentication/authorization stack: `Subject`/`Principal`
  concepts, a pluggable **login module** chain (`security/login`), two concrete login modules
  (`security/password`, `security/jwt`), and a declarative **authorization** policy
  (`security/authorizes`).
- `validates/` — a validation **port** (`validates/it.go`, `filter.go`, `outcome.go`) with two
  interchangeable **adapters**: `validates/go` (wraps `github.com/asaskevich/govalidator`) and
  `validates/play` (wraps `github.com/go-playground/validator/v10`).

**Module note (2026-07-31, Module Organization Pass):** `security/jwt` (incl. `security/jwt/jwks`),
`validates/play`, and `validates/go` are each now their **own nested Go module**
(`require github.com/miruken-go/miruken v0.32.1`), split out of root the same way
`api/http/httpsrv/openapi`/`es/goes` were, since each pulls dependencies (JWT/JWKS libraries,
go-playground's validator+translator, govalidator respectively) that only apps actually using
that specific feature need. Import paths are unchanged; nothing else in this doc changes as a
result. `security/password` and `security/login` (the port) stay in root — no heavy
third-party deps of their own.

## 2. Security: Subject & Principal

- `security.Principal` — `interface { Name() string }`. Minimal identity marker.
- `security.Subject` (`security/subject.go:17`) — mutable holder of `Principals()` and
  `Credentials()`, with `Authenticated()` derived from having either. Concrete impl
  `mutableSubject`; construct via `security.NewSubject(opts ...SubjectOption)` with
  `WithPrincipals`/`WithCredentials` functional options.
- `security.System` / `security.SystemSubject` (`subject.go:156`) — a singleton immutable
  "bypass" subject/principal for service-to-service calls; `authorizes` filter special-cases it
  (`principal.All(subject, security.System)` skips all checks — see §3).
- `security/principal` package supplies concrete string-based `Principal` types: `Id`, `User`,
  `Email`, `Role`, `Group`, `Entitlement` — all `~string` types implementing `Name()` **and**
  `InitWithTag(tag reflect.StructTag)` reading a `name:"..."` tag. This means any of them can
  also be used as a **Constraint** in a policy-marker struct (e.g.
  `principal.Role \`name:"manager"\`` inside an annotation struct), reusing the same tag-driven
  init mechanism `constraint.go`'s `Named`/`Metadata` use.
- Helpers: `principal.All(subject, ps...)`, `principal.Any(...)`, `principal.First[T](subject)`,
  `principal.Find[T](subject)`, `principal.Parse[T StringPrincipal](val any)` (parses
  string/[]string/[]any claim values into principals — used heavily by the JWT module, §4).

## 3. Security: Authorization (`security/authorizes`)

`authorizes.It` (`security/authorizes/it.go`) is its own **contravariant Callback** (own
`ContravariantPolicy` instance, independent of `handles`), carrying an `action any` as its
`Source()`/`Key()`. Handler methods declare authorization logic exactly like a `handles.It`
handler, just keyed off `*authorizes.It`:

```go
// security/authorizes/test/authorizes_test.go
func (t *TransferFundsAccessPolicy) AuthorizeTransfer(
    _ *authorizes.It, transfer TransferFunds,
    subject security.Subject,
) bool {
    if transfer.Amount < 10000 { return true }
    return principal.All(subject, principal.Role("manager"))
}

// constrained variant — only matches Access() calls passing the "fast" constraint
func (t *TransferFundsAccessPolicy) AuthorizeTransferFast(
    _ *struct {
        authorizes.It
        constraint.Named `name:"fast"`
    }, transfer TransferFunds,
    subject security.Subject,
) *promise.Promise[bool] { ... }
```

- `authorizes.Access(handler, action, constraints...) (bool, *promise.Promise[bool], error)`
  drives the check: `handler.Handle(auth, false, nil)`; if **not handled** and
  `authorizes.Options{RequirePolicy: true}` is set (via `miruken.Options`), access is denied;
  otherwise unhandled defaults to **allow**. This permissive default is intentional and scoped
  to this free function only — it's for ad-hoc `Access()` calls in an app that may not have any
  authorization policy wired up at all (see `Default`/`RequiresPolicy` in
  `security/authorizes/test/authorizes_test.go`).
  **Fixed 2026-07-31**: this permissive default used to also apply, indirectly, to the
  `authorizes.Required` filter below — meaning a `Required`-protected method with no matching
  `AuthorizeX` policy anywhere would silently allow access instead of denying it. `Access`'s
  logic was split into an unexported `access(handler, action, requirePolicy bool,
  constraints...)`; `Access` still passes the configured `Options.RequirePolicy`, but
  `filter.Authorize` (below) now always passes `requirePolicy=true`, so a binding that
  explicitly declares `Required` fails closed on an unhandled check regardless of the ambient
  `Options`. See `security/authorizes/it.go` and `filter.go`, and the
  `Filter/DeniedWithoutPolicy` test case.
- Enforcement side: `authorizes.Required` (`security/authorizes/filter.go`) is a
  `FilterProvider` + `Constraint`-like marker consumed the same way `validates.Required` and
  `Routes` are (§ pattern shared with `api/route.go`) — embed it in a handler method's policy
  struct to require authorization before the method runs:

```go
func (a *Account) Transfer(
    _ *struct {
        handles.It
        authorizes.Required
    }, transfer TransferFunds,
) Money { ... }
```

  Its filter (`filter.Authorize`, order = `miruken.FilterStageAuthorization`) resolves the
  ambient `security.Subject` (via DI — dependency-injected as a `Filter.Next` parameter,
  `subject security.Subject`), skips checks for `principal.All(subject, security.System)`,
  checks `binding.Metadata()` for any `security.Principal` markers (`checkBindingPrincipals`,
  filter.go:118 — lets you tag a binding with a required principal via metadata rather than
  writing authorization code), then calls `authorizes.Access` recursively against the composer.
  Denial raises `*authorizes.AccessDeniedError` (sync) or `panic`s it inside a promise
  continuation (async) — the panic is caught upstream by the filter pipeline's promise
  machinery (see `core-dispatch.md` for how panics inside filters become results).
- `Required.InitWithTag` reads a `policy:"..."` tag to scope which named policy variant
  (`AuthorizeTransferFast` above) governs a given protected method — this is the "annotation
  argument" pattern again, tag value flows into the FilterProvider instance at setup time.

## 4. Security: Login Modules

`security/login` is a **JAAS-style pluggable authentication chain**:

- `login.Module` (`module.go`) — port: `Login(subject, handler) error`, `Logout(subject,
  handler) error`. A module can use the supplied `Handler` to *prompt* for credentials by
  dispatching its own callback types (`security/login/callback.Name`, `.Password`) — the actual
  answering is application-supplied via `NameHandler{Name: "..."}` /
  `PasswordHandler{Password: [...]}` simple `Handle()` implementations wired into the handler
  chain. This decouples "how a module asks for a credential" from "where the answer comes from"
  (CLI prompt, HTTP header, config, test double, etc.) — genuinely clean port/adapter split.
- `login.Configuration` / `login.Flow` / `login.ModuleEntry` (`config.go`) — declarative,
  config-driven module chains: `map[string]Flow` where each `Flow` is an ordered list of
  `{Module string, Options map[string]any}`. Modules are resolved by *key* via
  `creates.Key[Module](handler, entry.Module)` (DI-style, string-keyed creation — see
  `core-dispatch.md`'s `creates` policy) so login flows are wired from config
  (`config.Load{Path: flowAlias}`, integrating with `config/` — see support-libs doc) rather
  than code.
- `login.Context` (`context.go`) — orchestrates one login attempt: builds `modules` lazily
  (`initFlow`), calls `Login` on each in order; on failure, calls `Logout` on the
  already-succeeded modules in reverse (rollback semantics), wraps failures in `login.Error`.
  Returns `*promise.Promise[security.Subject]` — the whole flow is async-capable end to end.
- Both concrete modules follow the same constructor idiom, keying their `creates.It` binding by
  a string so `creates.Key` can find them by name from config:

```go
// security/password/login.go
func (l *LoginModule) Constructor(
    _ *struct{ creates.It `key:"login.pwd"` },
    verifiers []Verifier,
) { l.verifiers = verifiers }

// security/jwt/login.go
func (l *LoginModule) Constructor(
    _ *struct{ creates.It `key:"login.jwt"` }, jwks KeySet,
) { l.jwks = jwks }
```

  This is the `addPolicy` path in `bind.go` using the `key:"..."` struct tag
  (`bindingSpec.addPolicy`, `bind.go:150`) to register an **invariant, string-keyed** binding
  alongside the type-keyed one — i.e. the same constructor can be found either by type or by
  the login-flow config string.
- `password.LoginModule` — `Init(opts map[string]any) error` parses YAML/JSON-shaped config
  (`credentials: {user: pass}` map or list-of-maps form) into a `password.Map` `Verifier`;
  `Verifier` is itself a port (`VerifyPassword(username, password) bool`) so credential storage
  is swappable (map today, presumably DB/LDAP adapters later — none present yet).
- `jwt.LoginModule` — `Init` parses `issuer`/`audience`/`jwks.uri`|`jwks.keys` options; `Login`
  parses+verifies the JWT via `golang-jwt/jwt/v5`, then maps claims to `security.Principal`s
  (`sub`→`principal.Id`, `scp` space-delimited scopes→`jwt.Scope`, `email`/`role(s)`/`group(s)`/
  `entitlement(s)`→corresponding `principal.*` via `principal.Parse[T]`). Delegates actual key
  resolution to `jwt.KeySet` port (`At(uri)`/`From(json)`), implemented by `security/jwt/jwks`
  (JWKS fetch/cache — not read in this pass, but the port boundary is clean: `LoginModule` never
  touches HTTP or JSON-key-parsing directly).

## 5. Validation: the Port

- `validates.It` (`validates/it.go`) — its own **contravariant Callback**, `Source()` is the
  value being validated, carries a `groups []any` (for group-scoped validation, JSR-303 style)
  and an `Outcome` accumulator (`Outcome() *Outcome`, mutated in place by handlers).
- `validates.Group` — a `Constraint` (implements `Required()/Implied()/InitWithTag/Satisfies`
  and also a `Merge(Constraint) bool` — merges multiple `Groups(...)` markers on the same
  binding into one set) reading a `name:"group"` tag; `"*"` (`anyGroup`) matches everything.
  Build one directly with `validates.Groups(groups...)`.
- `validates.Outcome` (`outcome.go`) — structured, **path-addressable** error tree:
  `AddError(path, err)`, `FieldErrors(path)`, `Path`/`RequirePath` support dotted/indexed paths
  (`"addr.city"`, `"items[0].sku"`) and nest child `*Outcome`s recursively (so a struct
  validating a sub-struct field gets a sub-`Outcome`, and `Error()` renders nested outcomes as
  `"field: (nested: msg)"`). `Valid()` is simply `errors == nil`. This is a hand-rolled
  structured-error concept independent of both backend libraries — the two adapters (§6)
  translate their own error shapes into this common `Outcome`, so callers never see
  govalidator/play-validator types.
- `validates.Constraints(handler, source, constraints...) (*Outcome, *promise.Promise[*Outcome],
  error)` — the entry point application code calls to validate any value; dispatches an `It`
  callback greedily (`true`) so *every* matching validator handler runs and contributes to one
  shared `Outcome`.
- `validates.Required` (`filter.go`) — the enforcement-side `FilterProvider`, embedded the same
  way as `authorizes.Required`: put it in a handler's policy struct to auto-validate the
  callback's `Source()` before dispatch (and, if `validates:"output"` tag set, the return value
  after). `AppliesTo` restricts it to `*handles.It` callbacks with a non-nil source. Filter order
  = `miruken.FilterStageValidation`. Handles sync and async (promise) input/output validation
  uniformly, aborting the pipeline (returning the `*Outcome` as the error) on invalid input, or
  panicking the outcome inside a promise continuation for async output validation (caught
  upstream, same pattern as `authorizes`).
- `validates.Feature()` (`feature.go`) — the port's `setup.Feature`: registers the `Required`
  filter globally (`b.Filters(&Required{i.output})`) so *every* binding across the app gets
  validation-filter wiring available; `validates.Output(installer)` config toggles output
  validation. Note this Feature does **not** install any concrete validator — it only wires the
  filter/port. An adapter package must be installed too (§6) or nothing actually validates
  (unhandled `It` dispatch simply returns `Outcome{}` which is `Valid()`).

## 6. Validation: Adapters

| | `validates/go` (govalidator) | `validates/play` (go-playground/validator) |
|---|---|---|
| Wraps | `github.com/asaskevich/govalidator` | `github.com/go-playground/validator/v10` |
| Handler type | `validator{}` (single, unexported) | `validator{ Validator }` **plus** `Validator`/`Validates[T]`..`Validates9[T]` generic families |
| Rule source | struct tags only (govalidator's own tags) | struct tags **or** code-defined `Rules`/`Constraints` (tag-free), registered via `RegisterStructValidationMapRules` |
| i18n | none | optional `ut.Translator` (`go-playground/universal-translator`), injected via DI (`args.Optional`) |
| Feature | `govalidator.Feature()`, `DependsOn() → validates.Feature()` | `playvalidator.Feature()`, `DependsOn() → validates.Feature()`, exposes `Validator()`/`UseTranslator()` on the `Installer` for pre-registering custom rules before `Feature()` builds the handler graph |
| Registration | `b.Specs(&validator{})` | `b.Specs(&validator{}).With(i.validate)` (+ optional translator) |

Both adapter handler methods have the identical shape:

```go
func (v *validator) Validate(it *validates.It, target any) miruken.HandleResult {
    if !internal.IsStruct(target) { return miruken.NotHandled }
    // ...call into the wrapped library, translate its errors into validates.Outcome...
    return miruken.HandledAndStop // or miruken.Handled
}
```

i.e. **both are ordinary `handles`-policy-shaped handlers keyed on `*validates.It`** — nothing
validation-specific in the core framework knows about either library; they are pure
peer adapters behind the `validates.It`/`Outcome` port.

The `play` adapter additionally demonstrates a documented Go workaround
(`playvalidator/validator.go:38-46`) for the lack of method overloading: to let **multiple**
type-specific validators coexist on one composed struct, it defines `Validates[T]`,
`Validates1[T]` … `Validates9[T]` as distinct generic types each embedding `Validator` and each
exposing a differently-named `Validate`, `Validate1`, … `Validate9` method — because Go
embedding silently drops a promoted method if two embedded types define the *same* method name,
so each numbered variant sidesteps that collision when an app composes several typed validators
into one handler struct.

## 7. Case study — port/adapter cleanliness assessment

**Validation is the cleanest hexagonal example in the repo.** The port (`validates.It`,
`Outcome`, `Required` filter, `Feature()`) has zero imports of either backend library. Both
adapters depend inward on `validates` and outward on their own library only; they never
reference each other. `validates.Feature()` wires the *filter*, while each adapter's `Feature()`
wires the *handler* and declares `DependsOn() → validates.Feature()` — so installing either
adapter transitively pulls in the port's plumbing, and an application can swap `govalidator.
Feature()` for `playvalidator.Feature()` (or install both, since both key off the same
`*validates.It` callback and Go's greedy dispatch just runs all matching handlers) with **no
consumer code changes**. The only mild leak: `validates.Outcome.Error()` special-cases
`err.(*Outcome)` for nesting (outcome.go:105), meaning the port has a hard assumption baked in
that nested-value errors are themselves `*Outcome` — acceptable since it's the port's own type,
not a backend leak.

**Authorization and login are good but not as pure.** `authorizes` is a fully self-contained
policy (own `It`, own filter, own `Options`) with no adapter/port split needed — there's only
one implementation strategy (application-supplied handler methods), so hexagonal separation
doesn't really apply here; it's more directly analogous to `handles` itself. `login.Module` *is*
a genuine port with two adapters (`password`, `jwt`) that never reference each other and depend
only on `security` + `login/callback`. The one thing worth flagging: `jwt.LoginModule` reaches
directly into `golang-jwt/jwt/v5` types (`jwt.Token`, `jwt.MapClaims`, `jwt.Keyfunc`) as its
*credential* representation stored on `security.Subject` (`subject.AddCredentials(l.token)`) —
a consumer inspecting a `Subject`'s credentials to do something claim-related is coupled to the
`golang-jwt` library even though `security.Subject.Credentials() []any` is nominally
library-agnostic. This is a minor, arguably acceptable leak (the credential *is* a JWT, so
representing it as one is honest) but worth knowing if you ever want to swap JWT libraries.
