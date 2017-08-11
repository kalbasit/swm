package main

import (
	"flag"
	"log"
	"os"
	"path"
	"regexp"

	"github.com/kalbasit/swm/code"
)

var (
	// codePath is the path to the code folder
	codePath string
)

func init() {
	flag.StringVar(&codePath, "code", path.Join(os.Getenv("HOME"), "code"), "The code path to scan")
}

func main() {
	// parse the flags
	flag.Parse()
	// create the code and scan
	c := code.New(codePath, regexp.MustCompile("^.snapshots$"))
	// scan the code folder
	c.Scan()
	// save it
	if err := c.Save(); err != nil {
		log.Fatalf("error saving the code to the cache file: %s", err)
	}
}
