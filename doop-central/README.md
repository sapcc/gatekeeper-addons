## doop-central

Takes in audit reports submitted by multiple instances of
[doop-agent](../doop-agent/) and presents them as a web dashboard.

## Usage

The central itself is completely stateless, but some configuration must be provided.
Two command-line argument must be given:

```bash
$ doop-central 0.0.0.0:8080 docs.yaml
```

The first argument is the listen address for the HTTP server where the web
dashboard is exposed. The second argument is the path to a YAML file containing
human-readable descriptions for the policy checks reported on this dashboard, like so:

```yaml
GkImagesFromDockerHub: |
  <p>The following containers use an image from Docker Hub. You should pull
  from our internal registry at <code>registry.example.org</code> instead.</p>
```

Each key is a kind from the `constraints.gatekeeper.sh` API group, and the
value is a plain HTML string that will be shown at the top of the respective
dashboard section within a `<blockquote>`. In addition, the key `Header` is
allowed. The HTML in this key will be shown at the very top of the dashboard.

### Swift API

For downloading reports from Swift, OpenStack credentials must be present in the
usual `OS_...` environment variables. Furthermore, the `REPORT_CONTAINER_NAME`
environment variable must be given to identify where reports were uploaded
within the Swift account.

### HTTP API

The dashboard is exposed on the endpoint `GET /` for consumption by regular
browsers. Furthermore, a health check endpoint is provided at `GET
/healthcheck`, which always returns the plain text string "OK".
