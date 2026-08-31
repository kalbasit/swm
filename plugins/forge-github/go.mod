module github.com/kalbasit/swm/plugins/forge-github

go 1.26.7

replace (
	github.com/kalbasit/swm/proto => ../../proto
	github.com/kalbasit/swm/sdk/go => ../../sdk/go
)

require (
	github.com/google/go-github/v88 v88.0.0
	github.com/kalbasit/swm/proto v0.0.0
	github.com/kalbasit/swm/sdk/go v0.0.0
	github.com/pelletier/go-toml/v2 v2.4.3
	github.com/stretchr/testify v1.12.1
	google.golang.org/grpc v1.83.2
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

require (
	github.com/fatih/color v1.19.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-github/v90 v90.0.0
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-plugin v1.8.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/oklog/run v1.2.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260825221802-da73d73af1c5 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)
