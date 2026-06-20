# helm-release-size-analyser architecture guide

## 1. Purpose

`helm-release-size-analyser` is a local, side-effect-free CLI that estimates and explains the Kubernetes Secret produced when Helm installs a chart.

The binary must:

1. load a Helm chart and values using Helm-compatible behavior;
2. execute the Helm install action through the Helm Go SDK;
3. direct chart resource operations to an isolated in-memory Kubernetes substitute;
4. persist the release through Helm's real Secret storage driver into an in-memory Kubernetes client;
5. retrieve the resulting release Secret rather than reconstructing it independently;
6. decode the persisted release JSON and report its exact total size and the exact size of each top-level property;
7. never contact or mutate a user's Kubernetes cluster.

The first implementation should target Helm 4. Helm 4 is the current stable major version, already uses `log/slog` internally, and has breaking SDK changes compared with Helm 3. Pin an exact Helm minor version in `go.mod`; do not attempt to support both major versions behind one implementation.

## 2. Architectural decision: split the Kubernetes mock at Helm's two boundaries

Helm's action configuration has two distinct Kubernetes-facing dependencies:

- `KubeClient`, a `kube.Interface` used to build, create, update, delete, and optionally wait for rendered chart resources;
- `Releases`, a `storage.Storage` used to create and update Helm release records. With the Secret driver, this ultimately uses a typed Kubernetes `SecretInterface`.

Use a separate in-memory implementation for each boundary.

### Recommended design

| Boundary | Implementation | Responsibility |
| --- | --- | --- |
| Chart resources | An analyser-owned implementation of Helm's `kube.Interface` | Parse and record resource operations, return deterministic success, and perform no network I/O |
| Helm release storage | Helm's real Secret storage driver backed by `k8s.io/client-go/kubernetes/fake` | Exercise Helm's real release encoding, compression, Secret naming, labels, and persistence |

The storage path should conceptually be:

`action.Install` → `storage.Storage` → Helm Secret driver → fake CoreV1 Secret client → object tracker

After installation, retrieve the Secret from the fake client using its expected release identity, then independently verify that exactly one matching revision exists. Do not use Helm's memory storage driver: it stores a release object but does not create the Secret that this tool exists to analyse.

The chart-resource recorder should retain the rendered resources and operations for diagnostics, but it should not be responsible for synthesizing the Helm release Secret. Keeping these roles separate ensures the measured Secret is created by Helm production code.

### Why `client-go/fake` alone is insufficient

`fake.NewClientset`/`fake.NewSimpleClientset` is an object-tracker-backed typed client. It does not expose an HTTP Kubernetes API endpoint, and Helm's normal Kubernetes client builds REST clients, discovery clients, REST mappings, and resource visitors from a `RESTClientGetter`. It therefore cannot simply be passed to `action.Configuration.Init` as a replacement cluster.

The fake client is still a good fit for release storage because the Helm Secret driver consumes the narrow typed Secret client interface. Current client-go documentation explicitly describes its fake as a unit-test client without server-side validation, defaults, or field management.

### Higher-fidelity alternative: `envtest`

Use controller-runtime `envtest` only as an optional integration-test backend, not as the default runtime architecture. It starts real `kube-apiserver` and etcd binaries, which gives discovery, REST mapping, API validation, generated metadata, and realistic Secret admission behavior. The costs are binary downloads, process startup, filesystem use, platform constraints, and no built-in workload controllers. Charts that wait for Deployments, Jobs, or hooks can still stall because no controller reconciles them.

This leads to a useful test split:

- unit and normal CLI execution: injected Helm `kube.Interface` recorder plus client-go fake Secret client;
- selected compatibility tests: `envtest`, with waiting disabled and required CRDs installed explicitly;
- optional end-to-end tests: a disposable real cluster such as `kind`, outside the normal binary.

Other candidates are weaker defaults:

- controller-runtime's fake client has the same broad class of missing API-server behavior and does not directly satisfy Helm's REST-based resource path;
- an `httptest` server that emulates Kubernetes would require maintaining discovery, REST mapping, patch/apply, watch, status, and admission semantics;
- `kind`, k3s, and similar distributions are realistic but are not in-memory mocks and substantially increase runtime and operational cost.

## 3. Runtime flow

1. Cobra parses the command and flags.
2. A command-local Viper instance merges defaults, config file, environment, and flags into a typed immutable configuration.
3. Validation rejects invalid input before chart loading.
4. The application creates a request-scoped `slog.Logger` and the two isolated Kubernetes dependencies.
5. The chart source component resolves and loads a local chart directory or packaged chart. Remote repository and OCI acquisition should be a later, isolated extension.
6. The values component merges one or more values files and `--set`-style overrides with Helm-compatible parsers.
7. The Helm adapter constructs `action.Configuration` explicitly rather than calling its cluster-oriented `Init` method:
   - set the analyser's `kube.Interface` recorder;
   - initialize Helm `storage.Storage` with its Secret driver and the fake Secret client;
   - provide deterministic default Kubernetes capabilities;
   - pass the configured `slog.Handler` through Helm's logger hook;
   - avoid a real `RESTClientGetter` unless a supported chart feature requires it.
8. The adapter creates `action.Install`, sets release name and namespace, disables waiting and atomic behavior, and executes the context-aware install path.
9. The Secret collector lists the fake client's Secrets by Helm ownership/name/revision labels, validates the result, and returns a deep copy.
10. The analyser decodes the release JSON from the persisted Secret and measures those exact bytes. It does not analyse the SDK-returned release object.
11. The renderer writes table or JSON output to stdout. Logs and diagnostics go to stderr.

Every component must accept `context.Context`. Cancellation should stop chart acquisition, installation, and output work.

## 4. Helm install semantics

The install should be a real Helm action, but its supported semantics must be deliberately constrained:

- `DryRun` must be disabled. A client dry run can skip release persistence, which would defeat Secret capture.
- `Wait`, `WaitForJobs`, and `Atomic` must default to false because the in-memory resource recorder has no controllers.
- pre/post-install hooks should be disabled initially. Hook execution frequently depends on Pod/Job status and logs. Add support only with explicit simulated semantics.
- namespace creation should be unnecessary; the fake storage client is namespace-scoped and the resource recorder can accept Namespace objects. The requested release namespace is still recorded accurately.
- cluster lookups from templates, DNS template functions, schema validation against live discovery, server-side apply behavior, and generated-name behavior cannot be claimed as API-server-faithful in mock mode.
- default Kubernetes capabilities must be explicit and reported in the result because `.Capabilities` can change rendered output and therefore release size. Later flags may select a Kubernetes version and API-version set.
- CRDs should be included in the release exactly as Helm normally records them, while their mock application is recorded without pretending to establish a real API.

The analyser reports only sizes from the release JSON actually persisted by Helm's Secret driver. It does not add warnings or estimates derived from the simulated resource client.

## 5. Analysis model

Release size is the byte length of the exact JSON obtained by decoding the payload in the Secret persisted by Helm's Secret driver. The report contains:

| Field | Meaning |
| --- | --- |
| `total_bytes` | Length of the complete decoded release JSON, including outer braces and all JSON syntax |
| `properties[].name` | Decoded name of a top-level release JSON property |
| `properties[].bytes` | Exact byte span of that property in the persisted JSON, including its encoded key, value, whitespace, and delimiter comma |

Property order matches the persisted JSON. Outer object braces contribute to `total_bytes` only. Values are never re-encoded for measurement, so escaped characters, spaces, quotes, keys, commas, and colons retain their actual stored size.

## 6. CLI design with Cobra and Viper

Start with one primary command:

`helm-release-size-analyser analyse CHART`

The root command should only own global concerns such as logging and config discovery. `analyse` owns chart, values, simulated cluster, threshold, and output options. Keep command construction side-effect free and inject an application service so CLI tests do not run Helm implicitly.

Suggested initial flags:

| Flag | Purpose |
| --- | --- |
| `--release-name` | Deterministic release identity; derive a safe default from chart name if omitted |
| `--namespace` | Simulated release namespace |
| `-f`, `--values` | Repeatable values files |
| `--set`, `--set-string`, `--set-file` | Helm-compatible value overrides |
| `--kube-version` | Capabilities used during rendering |
| `--api-versions` | Additional simulated APIs |
| `--include-crds` | Match Helm install behavior explicitly |
| `-o`, `--output` | `table` or `json` |
| `--config` | Explicit configuration file |
| `--log-level` | `debug`, `info`, `warn`, or `error` |
| `--log-format` | `text` or `json` |

Use `viper.New()` per command/application instance. Avoid the package-level Viper singleton because it leaks state across tests and concurrent invocations. Bind Cobra/pflag flags explicitly, configure an environment prefix such as `HELM_RELEASE_SIZE_ANALYSER`, replace dots/dashes consistently for environment keys, and unmarshal once into a typed configuration struct. Validate that struct and pass it downward; business packages must not read Viper directly.

Viper's documented precedence is explicit `Set`, flags, environment, config file, key/value stores, then defaults. Document the subset actually supported by this binary. A missing implicit config file may be ignored; an explicitly requested config file that cannot be read must be an error.

## 7. Logging with `log/slog`

Create one logger at the composition root and inject it. Do not use global logger mutation.

- Text logs are the human default; JSON logs are suitable for CI.
- Logs always go to stderr so JSON reports on stdout remain machine-readable.
- Add stable attributes such as `component`, `chart`, `release`, and `namespace` at component boundaries.
- Pass the logger's handler into Helm 4's configuration so Helm and application messages share level and format policy.
- Never log rendered Secret values or complete values maps. Log sizes, keys, paths, and object identities only.
- Errors should be returned with context; log them once at the command boundary.

## 8. Package boundaries

Keep `cmd/` thin and put behavior under `internal/`:

```text
cmd/helm-release-size-analyser/   process entry point only
internal/cli/                     Cobra commands, Viper loading, exit mapping
internal/config/                  typed configuration and validation
internal/chartsource/             local chart and values loading
internal/helminstall/             Helm action adapter and configuration assembly
internal/kubemock/                kube.Interface recorder and fake client composition
internal/releasesecret/           Secret lookup, validation, and decoding boundary
internal/analyse/                 metrics, attribution, thresholds, result model
internal/report/                  table/JSON rendering
internal/buildinfo/               version metadata
```

Important interfaces should follow external boundaries, not mirror every concrete type:

- installer: chart plus values to install result;
- release Secret source: release identity to Kubernetes Secret;
- analyser: install artifacts to report model;
- report writer: report model to output.

Keep Helm and Kubernetes types at adapter boundaries where they add value. The stable report model should be analyser-owned so output compatibility is not tied to upstream structs.

## 9. Error and exit contract

Use stable categories and map them at the CLI boundary:

| Exit code | Meaning |
| --- | --- |
| 0 | Analysis completed and configured policy passed |
| 1 | Invalid input, chart/values failure, Helm failure, mock incompatibility, or internal error |

The report may still be emitted for a policy failure. Do not emit a partial machine-readable report for an operational failure unless the schema explicitly marks it incomplete.

Errors from Helm should preserve their cause. Add the release phase and relevant object identity, but do not duplicate long rendered manifests in the default error message.

## 10. Testing strategy

### Unit tests

- configuration precedence and validation with fresh Viper instances;
- Cobra argument/flag behavior using injected streams;
- Secret selection by labels, name, namespace, and revision;
- exact byte metrics from fixed release JSON fixtures;
- deterministic report ordering and golden output;
- no secrets or values appearing in logs.

### Helm adapter tests

Use small fixture charts to verify that a real `action.Install` writes a Helm release Secret into the fake client. Cover dependencies, CRDs, hooks rejected/disabled, large files, binary files, schema errors, and unsupported lookup behavior. Assert the captured Secret identity and decoded JSON.

### Integration tests

Run a focused suite against `envtest` to compare captured Secret bytes and metadata with the default in-memory path for supported charts. Run a smaller disposable-cluster suite to detect differences introduced by API-server behavior. Pin Helm and Kubernetes dependency versions so fixture changes are intentional.

### Architectural invariants

Tests should prove:

- no kubeconfig is loaded in default mode;
- no network request is possible in default mode;
- the Secret originates from Helm's Secret driver, not analyser reconstruction;
- concurrent analyses do not share fake objects, Viper state, log state, or Helm storage;
- identical inputs and pinned dependency versions produce deterministic metrics, excluding explicitly normalized timestamps/metadata.

## 11. Delivery sequence

1. Establish the typed config, CLI contract, report schema, and exit policy.
2. Implement the fake Secret storage path and prove that Helm creates a retrievable Secret.
3. Implement the minimal chart-resource `kube.Interface` recorder needed by install.
4. Support local charts and values with waiting and hooks disabled.
5. Add exact Secret metrics and deterministic JSON output.
6. Add exact top-level property analysis and human-readable table output.
7. Add `envtest` comparison tests.
8. Consider remote/OCI chart sources only after the local analysis contract is stable.

The critical feasibility spike is steps 2–3. It should be completed before investing in detailed reports: Helm SDK interfaces can change between minor versions, and the project depends on keeping the production Secret encoder while replacing the resource client safely.

## 12. Sources and version assumptions

This guide was researched against the current documentation and Helm source available on 2026-06-20:

- [Helm 4 action configuration and storage initialization](https://github.com/helm/helm/blob/v4.2.0/pkg/action/action.go)
- [Helm 4 Kubernetes client](https://github.com/helm/helm/blob/v4.2.0/pkg/kube/client.go)
- [Helm SDK install example](https://helm.sh/docs/topics/advanced/)
- [client-go fake clientset documentation](https://pkg.go.dev/k8s.io/client-go/kubernetes/fake)
- [client-go testing fake and reactors](https://pkg.go.dev/k8s.io/client-go/testing)
- [controller-runtime envtest documentation](https://book.kubebuilder.io/reference/envtest)
- [Viper documentation](https://pkg.go.dev/github.com/spf13/viper)
- [Cobra documentation](https://cobra.dev/)
- [`log/slog` documentation](https://pkg.go.dev/log/slog)

Before implementation, select and pin mutually compatible Helm and Kubernetes module versions. Re-check the exact `kube.Interface`, Secret driver constructor, release encoding API, and context-aware install method against that pinned Helm release rather than relying on this guide as an API signature reference.
