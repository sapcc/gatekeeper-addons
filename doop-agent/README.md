## doop-agent

Runs in a Kubernetes cluster alongside a Gatekeeper instance. Every 5 minutes,
all template errors and audit violations are collected and pushed into a Swift
container for further processing by [doop-central](../doop-central/).

## Usage

The agent itself is completely stateless, but some configuration must be provided.

### Identity

A positional argument must be given to classify the cluster where the agent
is running. The value must be a `map[string]string` serialized as JSON, e.g.
`{"layer":"prod","type":"admin"}`. The set of keys should be consistent among
all agents that send reports into the same Swift container.

### Swift API

For uploading reports into Swift, OpenStack credentials must be present in the
usual `OS_...` environment variables. Furthermore, the options `-container` and
`-object` must be given to identify where the report is uploaded within the
Swift account.

### Kubernetes API

To gather audit data, the uplink needs read access to the Kubernetes API for:

- constraint templates (kind `ConstraintTemplate` in API group `templates.gatekeeper.sh`)
- constraint configs (all kinds in API group `constraints.gatekeeper.sh`)

By default, the agent expects to be running in a pod inside the same Kubernetes
cluster, so it will look for its service account token in the usual place. For
development purposes, the options `-kubeconfig` and `-context` can be used to
supply a configuration (like with kubectl).

## Metrics

A HTTP server is exposed providing a `/metrics` endpoint for Prometheus.
The listen address for this server defaults to `:8080`, and can be changed with
the `-listen` option.

| Metric | Description |
| ------ | ----------- |
| `doop_agent_report_submitted_at` | UNIX timestamp of when last report was submitted |
| `doop_agent_report_duration_secs` | how long it took to collect and submit the last report, in seconds |

Metrics are absent until the first report has been submitted.
