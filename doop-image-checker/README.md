# doop-image-checker

This small helper program provides an HTTP endpoint that Rego expressions can call via the
[`http.send` built-in](https://www.openpolicyagent.org/docs/latest/policy-reference/#http).
The endpoint takes a reference to an image stored in [Keppel](https://github.com/sapcc/keppel),
pulls the image and returns all the response headers from the manifest pull as JSON.

Runs in a Kubernetes cluster alongside a Gatekeeper instance.

## Usage

The helper itself is completely stateless. The only configuration for production is the listen
address for the HTTP server:

```bash
doop-image-checker 0.0.0.0:8080
```

For testing purposes a second argument can be added which points to a yaml file containing mappings from image refs to headers.
The headers `X-Keppel-Max-Layer-Created-At` and `X-Keppel-Min-Layer-Created-At` have special handling that they accept durations like `-1h`.

```bash
doop-image-checker 0.0.0.0:8080 response-config.yaml
```

`response-config.yaml`:

```yaml
keppel.example.com/vulnerability:medium:
  X-Keppel-Max-Layer-Created-At: "-1h"
  X-Keppel-Min-Layer-Created-At: "-1h"
  X-Keppel-Vulnerability-Status: Medium
keppel.example.com/vulnerability:old:
  # older than slightly over a year (~13 months)
  X-Keppel-Max-Layer-Created-At: "-10000h"
  X-Keppel-Min-Layer-Created-At: "-10000h"
```

## API

The HTTP endpoint for header checking is `GET /v1/headers?image=:image`, for instance:

```
GET /v1/headers?image=keppel.example.com/foo/bar:latest
```

For each request, the respective manifest is pulled and all response headers are returns as a JSON object with keys in HTTP's canonical title case, for example:

```json
{
  "Content-Type": "application/vnd.docker.distribution.manifest.v2+json",
  "Content-Length": "1367",
  "Docker-Content-Digest": "sha256:64278080eee0d697343d15735979ea8c1a9c3b330a5ac5195e6e713ea2f8b9ea",
  "Docker-Distribution-Api-Version": "registry/2.0",
  "X-Keppel-Vulnerability-Status": "Clean",
  ...
}
```

The checker may cache headers for a short period of time to avoid unreasonable
load on the Keppel API.

Additionally, a health check endpoint is provided at `GET /healthcheck`, which
always returns the plain text string "OK".

## Logging

HTTP requests are logged, but by default, only failed requests (HTTP status code
!= 200) are logged. To enable full logging, set the environment variable
`LOG_ALL_REQUESTS=true`.
