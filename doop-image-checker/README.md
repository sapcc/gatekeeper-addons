## doop-image-checker

This small helper program provides an HTTP endpoint that Rego expressions can call via the
[`http.send` built-in](https://www.openpolicyagent.org/docs/latest/policy-reference/#http).
The endpoint takes a reference to an image stored in [Keppel](https://github.com/sapcc/keppel)
and returns the image's vulnerability status string.

Runs in a Kubernetes cluster alongside a Gatekeeper instance.

## Usage

The helper itself is completely stateless. The only configuration is the listen
address for the HTTP server, which must be supplied as the only command-line
argument:

```bash
$ doop-image-checker 0.0.0.0:8080
```

## API

The HTTP endpoint for vulnerability checking is `GET /v1?image=:image`, for instance:

```
GET /v1?image=keppel.example.com/foo/bar:latest
```

For each request, the respective manifest is pulled and the value of the
`X-Keppel-Vulnerability-Status` header is returned. The checker may cache
vulnerability statuses for a short period of time to avoid unreasonable load on
the Keppel API.

Additionally, a health check endpoint is provided at `GET /healthcheck`, which
always returns the plain text string "OK".

## Logging

HTTP requests are logged, but by default, only failed requests (HTTP status code
!= 200) are logged. To enable full logging, set the environment variable
`LOG_ALL_REQUESTS=true`.
