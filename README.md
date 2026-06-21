[![CI](https://github.com/jkroepke/helm-release-size-analyzer/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/jkroepke/helm-release-size-analyzer/actions/workflows/ci.yaml)
[![GitHub license](https://img.shields.io/github/license/jkroepke/helm-release-size-analyzer)](LICENSE.txt)
[![Current Release](https://img.shields.io/github/release/jkroepke/helm-release-size-analyzer.svg?logo=github)](https://github.com/jkroepke/helm-release-size-analyzer/releases/latest)
[![GitHub Repo stars](https://img.shields.io/github/stars/jkroepke/helm-release-size-analyzer?style=flat&logo=github)](https://github.com/jkroepke/helm-release-size-analyzer/stargazers)
[![GitHub all releases](https://img.shields.io/github/downloads/jkroepke/helm-release-size-analyzer/total?logo=github)](https://github.com/jkroepke/helm-release-size-analyzer/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/jkroepke/helm-release-size-analyzer)](https://goreportcard.com/report/github.com/jkroepke/helm-release-size-analyzer)
[![codecov](https://codecov.io/gh/jkroepke/helm-release-size-analyzer/graph/badge.svg)](https://codecov.io/gh/jkroepke/helm-release-size-analyzer)

# helm-release-size-analyzer

⭐ Don't forget to star this repository! ⭐

<p align="center">
<img src="./docs/web.png" alt="Web View">
</p>

## About

`helm-release-size-analyzer` shows how much space a Helm release occupies and
attributes the decoded release JSON size to its top-level properties.

Large chart files and CRDs can push Helm's stored release toward the Kubernetes
1 MiB object-size limit. [Handling Large Files and CRDs in Helm and the 1MB
Release Limit](https://jkroepke.de/2026/02/handling-large-files-and-crds-in-helm-and-the-1mb-release-limit/)
explains the constraint, why CRD-heavy charts are especially affected, and
ways to reduce release size. This analyzer complements that guidance with a
local, property-level view of the decoded release payload before installation.
The decoded size isn't the size of the encoded Kubernetes Secret object.

## Features

- Measures the exact decoded JSON persisted in a Helm release Secret.
- Reports the total size and exact size of every top-level property.
- Uses Helm's chart loader, values handling, install action, and Secret driver.
- Support local chart directories and packaged charts.
- Support values files, `--set`, `--set-string`, and `--set-file`.
- Produces human-readable table output, machine-readable JSON, or an
  interactive web report.
- Prints the uncompressed release JSON for further inspection.

## Quick start

### Installation

Download an archive for your platform from
[GitHub Releases](https://github.com/jkroepke/helm-release-size-analyzer/releases),
or install the latest version from the source:

```shell
go install github.com/jkroepke/helm-release-size-analyzer/cmd/helm-release-size-analyzer@latest
```

### Analyze a chart

Only **local** chart directories and packaged charts are supported.

Use the interactive web report to inspect every nested object property and
array element:

```shell
helm-release-size-analyzer analyze ./my-chart
```

The command listens on a random `127.0.0.1` port, opens the report in the
default browser, and runs until you select **Stop server** or interrupt the
command. The terminal logs the URL so that you can open it manually if the
browser can’t be started. The report is self-contained, loads no remote
assets, and sends no release data outside the local process.

```shell
helm-release-size-analyzer analyze ./my-chart \
  --release-name example \
  --namespace default \
  --output table
```

Example table output:

```text
PROPERTY      SIZE
TOTAL         991.00 B
COMPRESSED    612.00 B
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
helm-release-size-analyzer analyze ./my-chart --output json
```

```json
{
  "properties": [
    {
      "name": "name",
      "bytes": 17
    }
  ],
  "total_bytes": 991,
  "compressed_bytes": 612
}
```

To inspect the decoded payload directly:

```shell
helm-release-size-analyzer release-json ./my-chart > release.json
```

## Size definition

`total_bytes` is the byte length of the complete decoded and decompressed
release JSON,
including outer braces and JSON syntax. A property size includes its encoded
key, value, whitespace, and delimiter comma. JSON report property order matches
the stored JSON; table properties are ordered by decreasing size. These reports
also include second-level properties of `chart`, such as `chart.values`.
The web report displays each size once, using B or kibibyte-based KB as
appropriate.

The analyzer measures the original bytes after removing Helm's storage
encoding. It doesn’t re-encode values, estimate the payload from rendered
resources, or measure the SDK-returned release object.

## Command-line flags

The CLI is configured only through command-line flags.

Common flags:

| Flag             | Description                                                 |
|------------------|-------------------------------------------------------------|
| `--release-name` | Release name; defaults to the chart name                    |
| `--namespace`    | Simulated release namespace; defaults to `default`          |
| `-f`, `--values` | Values file; may be repeated                                |
| `--set`          | Set a value with Helm syntax                                |
| `--set-string`   | Set a string value with Helm syntax                         |
| `--set-file`     | Set a value from a file                                     |
| `--include-crds` | Include CRDs in the stored manifest                         |
| `-o`, `--output` | Output format: `table`, `json`, or `web`; defaults to `web` |
| `--log-level`    | Log level: `debug`, `info`, `warn`, or `error`              |
| `--log-format`   | Log format: `text` or `json`                                |

Run `helm-release-size-analyzer analyze --help` for the complete command
reference.

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
