[![CI](https://github.com/jkroepke/helm-release-size-analyzer/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/jkroepke/helm-release-size-analyzer/actions/workflows/ci.yaml)
[![GitHub license](https://img.shields.io/github/license/jkroepke/helm-release-size-analyzer)](LICENSE.txt)
[![Current Release](https://img.shields.io/github/release/jkroepke/helm-release-size-analyzer.svg?logo=github)](https://github.com/jkroepke/helm-release-size-analyzer/releases/latest)
[![GitHub Repo stars](https://img.shields.io/github/stars/jkroepke/helm-release-size-analyzer?style=flat&logo=github)](https://github.com/jkroepke/helm-release-size-analyzer/stargazers)
[![GitHub all releases](https://img.shields.io/github/downloads/jkroepke/helm-release-size-analyzer/total?logo=github)](https://github.com/jkroepke/helm-release-size-analyzer/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/jkroepke/helm-release-size-analyzer)](https://goreportcard.com/report/github.com/jkroepke/helm-release-size-analyzer)
[![codecov](https://codecov.io/gh/jkroepke/helm-release-size-analyzer/graph/badge.svg)](https://codecov.io/gh/jkroepke/helm-release-size-analyzer)

# helm-release-size-analyzer

`helm-release-size-analyzer` shows how much space a Helm release occupies and
attributes the decoded release JSON size to its top-level properties.

It runs Helm with isolated in-memory Kubernetes dependencies and measures the
Secret written by Helm's real Secret storage driver. It does not require a
cluster, load kubeconfig, contact a Kubernetes API, or modify cluster state.

Large chart files and CRDs can push Helm's stored release toward the Kubernetes
1 MiB object-size limit. [Handling Large Files and CRDs in Helm and the 1MB
Release Limit](https://jkroepke.de/2026/02/handling-large-files-and-crds-in-helm-and-the-1mb-release-limit/)
explains the constraint, why CRD-heavy charts are especially affected, and
ways to reduce release size. This analyzer complements that guidance with a
local, property-level view of the decoded release payload before installation.
The decoded size is not the size of the encoded Kubernetes Secret object.

## Features

- Measures the exact decoded JSON persisted in a Helm release Secret.
- Reports the total size and exact size of every top-level property.
- Uses Helm's chart loader, values handling, install action, and Secret driver.
- Supports local chart directories and packaged charts.
- Supports values files, `--set`, `--set-string`, and `--set-file`.
- Produces human-readable table output or machine-readable JSON.
- Prints the uncompressed release JSON for further inspection.
- Keeps logs on stderr and report data on stdout.

## Quick start

### Installation

Download an archive for your platform from
[GitHub Releases](https://github.com/jkroepke/helm-release-size-analyzer/releases),
or install the latest version from source:

```shell
go install github.com/jkroepke/helm-release-size-analyzer/cmd/helm-release-size-analyzer@latest
```

### Analyse a chart

```shell
helm-release-size-analyzer analyse ./my-chart \
  --release-name example \
  --namespace default
```

Example table output:

```text
PROPERTY      SIZE
TOTAL         991.00 B
name          17.00 B
info          165.00 B
chart         598.00 B
chart.values  31.00 B
manifest      155.00 B
version       12.00 B
namespace     22.00 B
apply_method  20.00 B
```

Actual sizes depend on the chart, values, and pinned Helm version.

Use JSON output in automation:

```shell
helm-release-size-analyzer analyse ./my-chart --output json
```

```json
{
  "properties": [
    {
      "name": "name",
      "bytes": 17
    }
  ],
  "total_bytes": 991
}
```

To inspect the decoded payload directly:

```shell
helm-release-size-analyzer release-json ./my-chart > release.json
```

## Size definition

`total_bytes` is the byte length of the complete decoded release JSON,
including outer braces and JSON syntax. A property size includes its encoded
key, value, whitespace, and delimiter comma. Property order matches the stored
JSON. The report also includes second-level properties of `chart`, such as
`chart.values`.

The analyzer measures the original bytes after removing Helm's storage
encoding. It does not re-encode values, estimate the payload from rendered
resources, or measure the SDK-returned release object.

## Command-line flags

The CLI is configured only through command-line flags.

Common flags:

| Flag | Description |
| --- | --- |
| `--release-name` | Release name; defaults to the chart name |
| `--namespace` | Simulated release namespace; defaults to `default` |
| `-f`, `--values` | Values file; may be repeated |
| `--set` | Set a value with Helm syntax |
| `--set-string` | Set a string value with Helm syntax |
| `--set-file` | Set a value from a file |
| `--include-crds` | Include CRDs in the stored manifest |
| `-o`, `--output` | Output format: `table` or `json` |
| `--log-level` | Log level: `debug`, `info`, `warn`, or `error` |
| `--log-format` | Log format: `text` or `json` |

Run `helm-release-size-analyzer analyse --help` for the complete command
reference.

## Limitations

The in-memory resource client is not a Kubernetes API server. It does not
provide discovery, admission, generated metadata, controllers, server-side
validation, or API-faithful template lookups. Hooks, waiting, atomic
installation, and live OpenAPI validation are disabled.

Charts that require live cluster state, hook workloads, or readiness
controllers are outside the default execution model.

## Documentation

- [`DEVELOPER.md`](DEVELOPER.md) describes the internal design and test
  strategy.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) explains the contribution workflow and
  required DCO sign-off.
- [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md) defines community expectations.

## Contributing

Contributions are welcome. Read [`CONTRIBUTING.md`](CONTRIBUTING.md) before
opening a pull request. Every commit must include a DCO sign-off.

## Copyright and license

Copyright 2026 Jan-Otto Kröpke.

Licensed under the [Apache License, Version 2.0](LICENSE.txt).

## Open Source Sponsors

Thanks to all sponsors!

* [@hegawa](https://github.com/hegawa) (25$) onetime
* [@Zero-Down-Time](https://github.com/Zero-Down-Time) (25$) onetime
* [@k0ste](https://github.com/k0ste) (25$) onetime

## Acknowledgements

Thanks to JetBrains IDEs for their support.

<table>
  <thead>
    <tr>
      <th><a href="https://www.jetbrains.com/?from=jkroepke">JetBrains IDEs</a></th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>
        <p align="center">
          <a href="https://www.jetbrains.com/?from=jkroepke">
            <picture>
              <source srcset="https://www.jetbrains.com/company/brand/img/logo_jb_dos_3.svg" media="(prefers-color-scheme: dark)">
              <img src="https://resources.jetbrains.com/storage/products/company/brand/logos/jetbrains.svg" style="height: 50px">
            </picture>
          </a>
        </p>
      </td>
    </tr>
  </tbody>
</table>
