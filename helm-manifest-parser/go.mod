module github.com/sapcc/gatekeeper-addons/helm-manifest-parser

go 1.17

require (
	github.com/golang/protobuf v1.5.2
	github.com/sapcc/go-bits v0.0.0-20210518135053-8a9465bb1339
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/helm v2.17.0+incompatible
)

require google.golang.org/protobuf v1.26.0 // indirect
