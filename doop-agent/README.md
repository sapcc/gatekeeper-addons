## doop-agent

Runs in a Kubernetes cluster alongside a Gatekeeper instance. Every 5 minutes,
all template errors and audit violations are collected and pushed into a Swift
container for further processing by [doop-central](../doop-central/).

## Usage

The agent itself is completely stateless, but some configuration must be provided.

### Swift API

For uploading reports into Swift, OpenStack credentials must be present in the
usual `OS_...` environment variables. Furthermore, the `REPORT_CONTAINER_NAME`
and `REPORT_OBJECT_NAME` environment variables must be given to identify where
the report is uploaded within the Swift account.

### Kubernetes API

To gather audit data, the uplink needs read access to the Kubernetes API for:

- constraint templates (kind `ConstraintTemplate` in API group `templates.gatekeeper.sh`)
- constraint configs (all kinds in API group `constraints.gatekeeper.sh`)

## Metrics

A HTTP server is exposed providing a `/metrics` endpoint for Prometheus.
The listen address for this server must be provided as the only command line argument.

The only metric presented is `doop_agent_last_successful_report`, given as a
UNIX timestamp in seconds (or 0 if no successful report was sent yet).
