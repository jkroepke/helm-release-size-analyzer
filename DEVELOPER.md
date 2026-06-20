# Developer guide

This document describes the implemented structure of
`helm-release-size-analyzer`. See [`CONTRIBUTING.md`](CONTRIBUTING.md) for
environment requirements, required checks, commit sign-off, and pull-request
guidance.

## Purpose

`helm-release-size-analyzer` is a local CLI that estimates and explains the
Kubernetes Secret produced when Helm installs a chart. It executes Helm's
install action with in-memory Kubernetes dependencies, captures the Secret
written by Helm's production storage driver, and measures the decoded release
JSON.

Normal execution must not load kubeconfig, contact a Kubernetes API, or mutate
a cluster.

## Package layout

| Path | Responsibility |
| --- | --- |
| `cmd/helm-release-size-analyzer` | Process entry point, signal handling, and exit status |
| `internal/cli` | Cobra commands, flag handling, logging, and component orchestration |
| `internal/config` | Typed configuration and validation |
| `internal/helminstall` | Chart loading, values merging, Helm action setup, installation, and Secret selection |
| `internal/kubemock` | Network-free implementation of Helm's chart-resource client |
| `internal/releasesecret` | Helm release payload decoding and validation |
| `internal/analyze` | Exact byte measurement and the stable report model |
| `internal/report` | Table, JSON, and loopback web report rendering |
| `internal/version` | Build metadata populated by release tooling |

Keep the command package thin. Helm and Kubernetes types belong at adapter
boundaries; the report contract remains owned by `internal/analyze`.

## Runtime flow

1. Cobra parses `analyze CHART` or `release-json CHART`.
2. Cobra writes command-line flag values into `config.Config`, which the CLI
   validates before use.
3. The CLI creates a request-scoped `slog.Logger` that writes to stderr.
4. `internal/helminstall` loads the local chart and merges Helm-compatible
   values files and `--set` options.
5. Helm receives two isolated Kubernetes-facing dependencies:
   - `internal/kubemock.Recorder` handles rendered chart resources without
     network access;
   - Helm's Secret storage driver writes release state to a fresh client-go
     fake clientset.
6. Helm runs the install with hooks, waiting, atomic behavior, and live OpenAPI
   validation disabled.
7. The installer selects the expected release revision from fake Secret
   storage and returns a deep copy.
8. `internal/releasesecret` removes Helm's base64 and optional gzip encoding
   and validates the resulting JSON.
9. `internal/analyze` measures the original decoded bytes without re-encoding
   them.
10. Table, JSON, or release JSON output is written to stdout. Web output starts
    a loopback HTTP server on a random port and opens the default browser.

Every analysis owns its fake client, Helm storage, logger, and configuration
state. Do not introduce shared mutable package state.

## Architectural invariants

- Measurements come from the Secret persisted by Helm's real Secret driver,
  never from the SDK-returned release object or a reconstructed payload.
- Chart-resource handling and release storage remain separate. The resource
  mock must not synthesize the release Secret.
- Secret lookup is constrained by release identity and revision and must reject
  an unexpected number of matches.
- Waiting, hooks, atomic installation, and live OpenAPI validation remain
  disabled unless deterministic in-memory semantics are implemented and
  tested.
- Logs and diagnostics use stderr. Reports and raw release JSON use stdout.
- Web reports bind only to `127.0.0.1`, use no remote assets, and stop on
  context cancellation or a request from the report page.
- Rendered secrets and complete values maps must not be logged.
- Context cancellation is propagated whenever the underlying API accepts a
  context.
- Default execution remains network-free and independent of user Kubernetes
  configuration.

## Measurement contract

`compressed_bytes` is the length of the decoded Kubernetes Secret value at
`data.release`, including Helm's base64 encoding of the gzip payload.
This is the release-data size relevant to Kubernetes' 1,048,576-byte Secret
limit. `total_bytes` is the length of the exact decoded and decompressed release
JSON, including outer braces and JSON syntax. Each entry in `properties` reports
the exact byte span of one top-level property in persisted order, including its
encoded key, value, whitespace, and delimiter comma. Entries prefixed with
`chart.` apply the same measurement to each second-level property of `chart`.
Outer braces contribute only to the total.

The analyzer must not decode and re-encode property values for measurement;
doing so would change escaping, whitespace, and potentially field ordering.
The web report extends the same span measurement recursively to every object
property and array element. Its scalar previews do not affect measurement.
Array elements whose value is an object expose a display label when their
`name` property is a string. The label does not affect measurement. The web
report orders object properties and array elements by decreasing size.
It displays each size once, using B or kibibyte-based KB as appropriate.
The table report also orders properties by decreasing size without changing the
persisted order exposed by the report model and JSON output.

## Supported behavior and limitations

The current implementation loads local chart directories or packaged charts.
It supports values files, `--set`, `--set-string`, `--set-file`, optional CRD
inclusion, table output, JSON output, interactive web output, and raw decoded
release JSON output. The `analyze` command defaults to web output.

The in-memory resource client is intentionally not an API server. It does not
provide discovery, admission, generated metadata, controllers, server-side
validation, or API-faithful behavior for template lookups. Features that need
live cluster state, hook workloads, or readiness controllers are outside the
default execution model.

## Testing

Tests should protect behavior at the narrowest useful boundary:

- flag handling, validation, deterministic defaults, and isolation in
  `internal/cli` and `internal/config`;
- values files, `--set` variants, release-name handling, Helm Secret creation,
  and selection by ownership, name, namespace, and revision in
  `internal/helminstall`;
- fixture charts covering dependencies, CRDs, hooks, large or binary files,
  schema errors, and unsupported live-cluster behavior;
- compressed, uncompressed, missing, and malformed payloads in
  `internal/releasesecret`;
- exact byte attribution for whitespace, escaped keys, empty objects, nested
  values, and trailing data in `internal/analyze`;
- deterministic output, stream errors, unsupported formats, recursive web
  measurements, and loopback server shutdown in `internal/report`;
- command behavior, strict stdout/stderr separation, cancellation, contextual
  errors, and exit mapping in `internal/cli`.

Integration tests should prove that normal execution cannot load kubeconfig or
perform network I/O and that concurrent analysis operations do not share
mutable state.
Keep fixture charts small and focused on one behavior.

Run the validation commands documented in `CONTRIBUTING.md` before submitting
changes.
