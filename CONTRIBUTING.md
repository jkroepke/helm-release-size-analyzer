# Contributing

Thank you for contributing to `helm-release-size-analyzer`.

## Prerequisites

- Use the Go version declared in [`go.mod`](go.mod).
- Do not update the Go version as part of an unrelated change.
- Read [`DEVELOPER.md`](DEVELOPER.md) for the project structure and runtime
  design.

## Development

Keep changes focused and include tests for new behavior and bugfixes. The CLI
must remain local and side-effect free: normal operation must not load a
kubeconfig, contact a Kubernetes API, or mutate a cluster.

Run the following checks before submitting a change:

```shell
make fmt
make lint
make test
```

`make fmt` can update Go source and module metadata. Review those changes
before committing them.

If a check cannot run because dependencies are unavailable or network access
is restricted, state that explicitly in the pull request's testing section.

## Documentation

When writing documentation, follow the
[textlint-rule-terminology](https://github.com/sapegin/textlint-rule-terminology)
rule and its
[terminology ruleset](https://github.com/sapegin/textlint-rule-terminology/blob/master/terms.jsonc).

## Pull requests

A pull request should:

- explain the problem and the chosen solution;
- summarize the user-visible and internal changes;
- include tests appropriate to the change;
- report the result of `make fmt`, `make lint`, and `make test`;
- cite relevant files and lines when describing implementation details;
- avoid including unrelated formatting or dependency changes.

## Developer Certificate of Origin

Every commit must include a `Signed-off-by` trailer to certify agreement with
the [Developer Certificate of Origin](https://developercertificate.org/).
Create signed-off commits with:

```shell
git commit --signoff
```

The trailer must contain your real name and a reachable email address, matching
the commit author. If a commit is missing the trailer, amend it before opening
or updating the pull request:

```shell
git commit --amend --signoff --no-edit
```
