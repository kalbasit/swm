module github.com/kalbasit/swm/cmd/swm

go 1.26.2

require (
	github.com/adrg/xdg v0.5.3
	github.com/gofrs/flock v0.13.0
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-plugin v1.8.0
	github.com/kalbasit/swm/proto v0.0.0
	github.com/kalbasit/swm/sdk/go v0.0.0
	github.com/pelletier/go-toml/v2 v2.3.1
	github.com/spf13/cobra v1.9.1
	github.com/stretchr/testify v1.11.1
	golang.org/x/sys v0.42.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/kalbasit/swm/proto => ../../proto
	github.com/kalbasit/swm/sdk/go => ../../sdk/go
)
