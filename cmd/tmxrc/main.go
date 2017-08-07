package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path"
	"regexp"

	"github.com/kalbasit/tmx"
)

var (
	// codePath is the path to the code folder
	codePath string

	// cachePath is the absolute path of the cache file
	cachePath string
)

func init() {
	flag.StringVar(&codePath, "code", path.Join(os.Getenv("HOME"), "code"), "The code path to scan")
	flag.StringVar(&cachePath, "cache", path.Join(os.Getenv("HOME"), ".cache", "tmx.json"), "The path of the file used to store the cache")
}

func main() {
	// parse the flags
	flag.Parse()
	// create the code and scan
	c := &tmx.Code{
		Path:           codePath,
		ExcludePattern: regexp.MustCompile("^.snapshots$"),
	}
	c.Scan()
	// open the file
	f, err := os.Create(cachePath)
	if err != nil {
		log.Fatalf("error opening the cache file %q: %s", cachePath, err)
	}
	defer f.Close()
	// marshal it into JSON
	if err := json.NewEncoder(f).Encode(c); err != nil {
		log.Fatalf("error saving code as JSON at %q: %s", codePath, err)
	}
}
