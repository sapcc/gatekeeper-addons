<!--
SPDX-FileCopyrightText: 2025 SAP SE
SPDX-License-Identifier: Apache-2.0
-->

# doop-analyzer

Runs in a Kubernetes cluster alongside a Gatekeeper instance. Once a minute, all template errors and audit violations
are collected and pushed into a Swift container for further processing by [doop-central](../doop-central/).

This is the successor to doop-agent. The main difference is that it takes on some of the more computationally expensive
analysis steps that used to be performed by doop-central.

## Usage

To run the analyzer in a Kubernetes cluster, use

```bash
doop-analyzer run <config-file>
```

The `run` subcommand will gather Gatekeeper audit data from Kubernetes once per minute, process this audit data
according to the configured rules, and then upload the resulting report into Swift. Some additional subcommands are
available to execute parts of this chain manually:

- `doop-analyzer collect-once <config-file>` gathers Gatekeeper audit data once and prints the gathered data onto stdout
  as JSON.
- `doop-analyzer process-once <config-file>` reads the output of `collect-once` from stdin and applies the configured
  rules to them. The resulting report is printed to stdout as JSON instead of being uploaded to Swift.

These subcommands are intended for automated tests of analyzer configuration files. Report data fixtures can be gathered
with `collect-once`, and the effect of configuration on these fixtures can be tested with `process-once`.

### Configuration

The analyzer itself is completely stateless, but some configuration must be provided.

- For uploading reports into Swift, the `run` subcommand requires OpenStack credentials which must be present in the
  usual `OS_...` environment variables.
- The rest of the configuration is collected from a YAML configuration file whose path is given as positional argument
  after the subcommand.

The following fields are allowed in the YAML configuration file:

| Field | Type | Description |
| ----- | ---- | ----------- |
| `cluster_identity` | object of strings | A classification of the cluster where the agent is running. The set of keys should be consistent among all analyzers that send reports into the same Swift container. |
| `kubernetes` | object | When not running inside a Kubernetes cluster, this section must be filled to refer to a Kubernetes client configuration. |
| `kubernetes.kubeconfig` | string | Path to a kubectl configuration file. |
| `kubernetes.context` | string | If not empty, overrides the default context setting in the kubeconfig. |
| `metrics.listen_address` | string | Listen address for Prometheus metrics endpoint. Defaults to `:8080`. Only needed for `run`. |
| `merging_rules` | list of objects | A sequence of rules that will be applied to each violation in order to group similar violations together. [See below](#rule-based-rewriting) for details. Only needed for `run` and `process-once`. |
| `processing_rules` | list of objects | A sequence of rules that will be applied to each violation in order to normalize its attributes. [See below](#rule-based-rewriting) for details. Only needed for `run` and `process-once`. |
| `swift.container_name` | string | Name of Swift container in which to upload report. Only needed for `run`. |
| `swift.object_name` | string | Object name with which report will be uploaded in Swift. Only needed for `run`. |

### Kubernetes API permissions

To gather audit data, the analyzer needs read access to the Kubernetes API for:

- constraint templates (kind `ConstraintTemplate` in API group `templates.gatekeeper.sh`)
- constraints (all kinds in API group `constraints.gatekeeper.sh`)

## Processing pipeline

### Labels and annotations

When compiling a report of all audit data, we recognize the following specific labels and annotations on the level of
constraints (i.e. all objects within the API group `constraints.gatekeeper.sh`):

- The annotation `template-source` may contain a URL pointing to the location in source code management where the
  respective ConstraintTemplate is defined.
- The annotation `constraint-source` may contain a URL pointing to the location in source code management where the
  respective constraint is defined.
- The annotation `docstring` may contain a Markdown string that explains to users browsing the DOOP UIs how to interpret
  and fix violations reported for this constraint.
- The label `severity` may contain one of the following strings to indicate the severity of the policy violation:
  `error`, `warning`, `info`, `debug`. The severity can be used by the UI to order violations and apply styling, e.g.
  different background colors. Constraints with `spec.enforcementAction = "deny"` should probably have a severity of
  `error`. Violations for constraints with severity `debug` should be hidden by default and only shown when explicitly
  requested by the user.

### Object identity

For each violation, Gatekeeper only reports the object's kind, namespace and name; as well as the violation message. To
carry additional structured object attributes, doop-analyzer can recognize a JSON payload embedded as a prefix into the
message like this:

```json
{"key1":"value1","key2":"value2"} >> violation message
```

The `>>` separator is part of the recognized format. If a JSON payload like this is found in the message and if it
parses into the Go type `map[string]string` (i.e. it is a flat object with only string values), it and the `>>`
separator will be stripped from the message. For example, a violation like this:

```json
{
  "kind": "Pod",
  "namespace": "myapp",
  "name": "myapp-mariadb-74g62",
  "message": "{\"team\":\"myteam\",\"layer\":\"prod\"} >> no resource limit set"
}
```

will be parsed into this:

```json
{
  "kind": "Pod",
  "namespace": "myapp",
  "name": "myapp-mariadb-74g62",
  "message": "no resource limit set",
  "object_identity": {
    "team": "myteam",
    "layer": "prod"
  }
}
```

To ensure that the object identity is parsed correctly, the `>>` separator should not be included in any keys or values
inside the object identity's JSON payload.

As with the cluster identity, doop-analyzer does not care about the exact keys and values in the object identity (except
that object identity must be equal when merging violations, see below), but to make further processing easier, it's
probably a good idea to keep the set of keys consistent across all violations.

### Rule-based rewriting

Custom rules can be provided in the configuration in order to process violations based on regex matches. Rewriting
happens in two phases.

#### Processing rules

In the first phase, the **processing rules** from the configuration section `processing_rules` are applied to each
matching violation. Processing rules rewrite the original violation itself. As a basic example, consider rules that
apply to Helm releases: As of Helm 3, Helm releases are stored in Secret objects with the name schema
`sh.helm.release.v1.${NAME}.v${VERSION}`. This implementation detail can be hidden from the user-visible violations by
rewriting with the following processing rule:

```yaml
processing_rules:
  - match: { kind: Secret }
    replace:
      source:  name
      pattern: 'sh\.helm\.release\.v1\.(.*\.v\d+)'
      target:  { kind: Helm 3 release, name: '$1' }
```

Applying this processing rule to a violation like this:

```json
{
  "kind": "Secret",
  "name": "sh.helm.release.v1.foo-bar.v1",
  "namespace": "foo",
  "message": "I don't like this",
  "object_identity": {}
}
```

will yield the following result:

```json
{
  "kind": "Helm 3 release",
  "name": "foo-bar.v1",
  "namespace": "foo",
  "message": "I don't like this",
  "object_identity": {}
}
```

#### Rule syntax

Each processing rule may contain the following fields:

| Field | Type | Description |
| ----- | ---- | ----------- |
| `description` | string | A human-readable description of what this rule is about. This field is not interpreted by doop-analyzer at all, but it can be used for documentation purposes. Since it's a structured field instead of just a YAML comment, it is more likely to be preserved when editing rules in a specialized UI. |
| `match` | object of regexes | If given, the rule will only be applied if, for each key-value pair in this object, the violation has an attribute whose name matches the key and whose value matches the regex. (In the example above, the rule only applies to violations whose `kind` attribute matches the regex `Secret`.) See below for notes on attribute names and regex syntax. |
| `replace.source` | string | *Required.* The attribute name within the violation whose value will be matched for this rule's replacement. See below for notes on attribute names. |
| `replace.pattern` | regex | *Required.* The rule will apply if the value from the `replace.source` attribute matches this regex. (In the example above, the rule performs a replacement if its regex matches the violation's `name` attribute.) See below for notes on regex syntax. |
| `replace.target` | object of regexes | *Required.* For each key-value pair in this object, the violation's attribute whose name matches the key will have its value replaced with the value in this object, except that placeholders like `$1`, `$2` and so on are replaced by the respective capture groups from the `replace.pattern` match. (In the example above, the rule updates the violation's `name` and `kind` attributes if the regex matches.) See below for notes on attribute names. |

When a violation attribute name is expected, valid values include `kind`, `name`, `namespace` and `message`.
Furthermore, object identity fields can be accessed with the name syntax `object_identity.$FIELD` (for example,
`object_identity.type`).

Fields that are described as regex-typed accept regex strings using the [syntax defined by Go's stdlib regex
parser](https://golang.org/pkg/regexp/syntax/). The anchors `^` and `$` are implied at both ends of the regex, and need
not be added explicitly. To match any value, write `.*`. Empty regexes are not allowed because they usually don't do
what one expects. To explicitly match only the empty string, write `^$` instead.

#### Merging rules

In the second phase of rule-based rewriting, **merging rules** from the configuration section are applied to each
matching violation. Merging rules do not rewrite the original violation. Instead, they operate on a clone of the
violation to obtain a **violation pattern**. After applying all merging rules, violations with the same pattern are
merged into **violation groups** to deduplicate the violation report. Inside each violation group, only the pattern will
be shown in full. Each violation within the group becomes a **violation instance** that only displays the fields in
which it differs from the computed pattern.

A common usecase for merging is to group together violations that are reported on each Pod of a common owner (e.g. a
single DaemonSet or Deployment). For example, the following violations both report the same problem across the
pods of a DaemonSet:

```json
[
  {
    "kind": "Pod",
    "name": "foobar-8237g",
    "namespace": "default",
    "message": "no CPU request set"
  },
  {
    "kind": "Pod",
    "name": "foobar-as4f8",
    "namespace": "default",
    "message": "no CPU request set"
  },
  {
    "kind": "Pod",
    "name": "foobar-1gyc4",
    "namespace": "default",
    "message": "no CPU request set"
  }
]
```

Using the following merging rule:

```yaml
merging_rules:
- match: { kind: Pod }
  replace:
    source:  name
    pattern: '(.*)-[a-z0-9]{5}'
    target:  { name: '$1-<variable>' }
```

These violations are merged into a violation group like this:

```json
{
  "pattern": {
    "kind": "Pod",
    "name": "foobar-<variable>",
    "namespace": "default",
    "message": "no CPU request set"
  },
  "instances": [
    { "name": "foobar-8237g" },
    { "name": "foobar-as4f8" },
    { "name": "foobar-1gyc4" }
  ]
}
```

Merging rules have the same structure and behavior as processing rules. The only difference is that they transform the
violation pattern instead of the violation itself.

## Metrics

The `run` subcommand starts an HTTP server and provides a `/metrics` endpoint for Prometheus.

| Metric | Description |
| ------ | ----------- |
| `doop_analyzer_last_successful_report` | UNIX timestamp in seconds when last report was submitted. |
| `doop_analyzer_report_duration_secs` | How long it took to collect and submit the last report, in seconds. |
