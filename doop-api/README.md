<!--
SPDX-FileCopyrightText: 2025 SAP SE
SPDX-License-Identifier: Apache-2.0
-->

# doop-api

Takes in audit reports from multiple instances of [doop-analyzer](../doop-analyzer/) and presents them in a single API.

This is the successor to doop-central (which did the same with doop-agent reports).
The API endpoints of doop-api start in `/v2/` to distinguish it from its predecessor.

## Usage

The central itself is completely stateless, but some configuration must be provided in environment variables:

| Variable | Default | Explanation |
| -------- | ------- | ----------- |
| `DOOP_API_LISTEN_ADDRESS` | `:8080` | Listen address for the HTTP server where the API is exposed. |
| `DOOP_API_SWIFT_CONTAINER` | *(required)* | Name of the Swift container where reports were uploaded to. |
| `DOOP_API_OBJECT_IDENTITY_LABELS` | *(empty)* | Whitespace-separated list of keys whose values will be carried over from `object_identity` into the label set of the violation count metrics (see below). |
| `OS_...` | *(required)* | A full set of OpenStack auth environment variables, with permissions for reading from the Swift container. See [documentation for openstackclient][os-env] for details. |

[os-env]: https://docs.openstack.org/python-openstackclient/latest/cli/man/openstack.html

## API endpoints

### GET /v2/violations

Returns the full report with all violations, grouped as much as possible. The report can be filtered with the following query arguments:

| Query variable | Explanation |
| -------------- | ----------- |
| `cluster_identity.$KEY` | Only show violations in clusters where `cluster_identity[$KEY]` is equal to the provided value. |
| `object_identity.$KEY` | Only show violations for objects where `object_identity[$KEY]` is equal to the provided value. |
| `template_kind` | Only show violations of constraints whose template kind is equal to the provided value. |
| `constraint_name` | Only show violations of constraints whose name is equal to the provided value. |
| `severity` | Only show violations of constraints whose `severity` label is equal to the provided value. |

Each query variable can be given multiple times, in which case violations need to match any of the provided values.

### GET /metrics

Provides Prometheus metrics.

| Metric | Explanation |
| ------ | ----------- |
| `doop_raw_violations` | Number of raw violations, grouped by constraint, source cluster and selected object identity labels. |
| `doop_grouped_violations` | Number of violation groups, grouped by constraint, source cluster and selected object identity labels. |
| `doop_oldest_audit_age_seconds` | Data age for each source cluster. |

"Selected object identity labels" refers to those specified in `DOOP_API_OBJECT_IDENTITY_LABELS` (see above).
