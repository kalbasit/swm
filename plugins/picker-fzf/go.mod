module github.com/kalbasit/swm/plugins/picker-fzf

go 1.26.2

replace (
	github.com/kalbasit/swm/proto => ../../proto
	github.com/kalbasit/swm/sdk/go => ../../sdk/go
)

require (
	github.com/kalbasit/swm/proto v0.0.0
	github.com/kalbasit/swm/sdk/go v0.0.0-20260817011815-e91651f6ba06
	github.com/stretchr/testify v1.11.1
	google.golang.org/grpc v1.83.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-plugin v1.8.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/oklog/run v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260810153831-ec0a7760b754 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
