# helm-manifest-generator

A test helper that generates synthetic Helm release objects from a declaration file in YAML syntax such as this:

```yaml
metadata:
  name: newapp
  namespace: apps
  status: deployed
  version: 73

values:
  global:
    region: eu-de-1

owner-info:
  support-group: foo
  service: bar
  maintainers: Jane Doe, Max Mustermann
  helm-chart-url: https://example.com

items:
  - apiVersion: v1
    kind: ConfigMap
    metadata:
      name: newapp-config
    data:
      foo: bar
```

Call as `helm-manifest-generator < input.yaml > output.yaml`. The output is the YAML declaration of a Kubernetes Secret
object containing the Helm release.

We use this in our [Gatekeeper policy test suite][tests] whenever a test needs an entire Helm release as input.

[tests]: https://github.com/sapcc/helm-charts/tree/master/system/gatekeeper/tests
