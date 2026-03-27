// NOTE: github.com/ContinuumApp/continuum requires the main repo to update its go.mod module path.
// Until then, use a replace directive for local development:
// replace github.com/ContinuumApp/continuum => ../../../gitlab/continuum

module github.com/ContinuumApp/continuum-plugin-tvdb

go 1.26.0

require (
	github.com/ContinuumApp/continuum v0.0.0
	golang.org/x/time v0.14.0
	google.golang.org/grpc v1.75.1
	google.golang.org/protobuf v1.36.11
)
