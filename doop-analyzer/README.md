## doop-analyzer

Runs in a Kubernetes cluster alongside a Gatekeeper instance. Once a minute, all template errors and audit violations
are collected and pushed into a Swift container for further processing by [doop-central](../doop-central/).

This is the successor to doop-agent. The main difference is that it takes on some of the more computationally expensive
analysis steps that used to be performed by doop-central.

## Usage

To run the analyzer in a Kubernetes cluster, use

```
doop-analyzer run <config-file>
```

The `run` subcommand will gather Gatekeeper audit data from Kubernetes once per minute, process this audit data
according to the configured rules, and then upload the resulting report into Swift. Some additional subcommands are
available to execute parts of this chain manually:

- `doop-analyzer collect-once <config-file>` gathers Gatekeeper audit data once and prints the gathered data onto stdout
  as JSON.
- `doop-analyzer analyze-once <config-file>` reads the output of `collect-once` from stdin and applies the configured
  rules to them. The resulting report is printed to stdout as JSON instead of being uploaded to Swift.

These subcommands are intended for automated tests of analyzer configuration files. Report data fixtures can be gathered
with `collect-once`, and the effect of configuration on these fixtures can be tested with `analyze-once`.

### Configuration

The analyzer itself is completely stateless, but some configuration must be provided.

- For uploading reports into Swift, the `run` subcommand requires OpenStack credentials must be present in the usual
  `OS_...` environment variables.
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
| `merging_rules` | list of objects | A sequence of rules that will be applied to each violation in order to group similar violations together. [See below](#merging-rules) for details. Only needed for `run` and `analyze-once`. |
| `processing_rules` | list of objects | A sequence of rules that will be applied to each violation in order to normalize its attributes. [See below](#processing-rules) for details. Only needed for `run` and `analyze-once`. |
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
  different background colors. Constraints with `spec.enforcementAction = "dryrun"` should probably have a severity of
  `error`. Violations for constraints with severity `debug` should be hidden by default and only shown when explicitly
  requested by the user.

### Object identity

For each violation, Gatekeeper only reports the object's kind, namespace and name; as well as the violation message. To
carry additional structured object attributes, doop-analyzer can recognize a JSON payload embedded as a prefix into the
message like this:

```
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

### Processing rules

TODO

### Merging rules

TODO

## Metrics

A HTTP server is exposed providing a `/metrics` endpoint for Prometheus. The listen address for this server defaults to
`:8080`, and can be changed with the `-listen` option.

| Metric | Description |
| ------ | ----------- |
| `doop_analyzer_report_submitted_at` | UNIX timestamp of when last report was submitted |
| `doop_analyzer_report_duration_secs` | how long it took to collect and submit the last report, in seconds |

Metrics are absent until the first report has been submitted.
