package main

import "flag"

var (
	// profiles represents the profile we are scanning
	profile string

	// workspace is the workspace we are scanning
	workspace string
)

func init() {
	flag.StringVar(&profile, "profile", "", "select the profile to scan. Default to all profiles")
	flag.StringVar(&workspace, "workspace", "", "select the workspace to scan. Default to all workspaces")
}

func main() {
	flag.Parse()

}

func scan(dir string) {

}
