## doop-central

Takes in audit reports submitted by multiple instances of
[doop-agent](../doop-agent/) and presents them as a web dashboard.

## Usage

The central itself is completely stateless, but some configuration must be provided.
The only command-line argument must be the listen address for the HTTP server where the web dashboard is exposed:

```bash
$ doop-central 0.0.0.0:8080
```

### Swift API

For downloading reports from Swift, OpenStack credentials must be present in the
usual `OS_...` environment variables. Furthermore, the `REPORT_CONTAINER_NAME`
environment variable must be given to identify where reports were uploaded
within the Swift account.

### HTTP API

The dashboard is exposed on the endpoint `GET /` for consumption by regular
browsers. Furthermore, a health check endpoint is provided at `GET
/healthcheck`, which always returns the plain text string "OK".
