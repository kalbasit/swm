module github.com/kalbasit/swm/plugins/picker-fzf

go 1.26.2

replace (
	github.com/kalbasit/swm/proto => ../../proto
	github.com/kalbasit/swm/sdk/go => ../../sdk/go
)

require (
	github.com/kalbasit/swm/proto v0.0.0
	github.com/kalbasit/swm/sdk/go v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.81.1
)

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-plugin v1.8.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/oklog/run v1.1.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
