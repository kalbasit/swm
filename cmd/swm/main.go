package main

import "gopkg.in/urfave/cli.v2"

func main() {
	app := &cli.App{
		Name:    "swm",
		Version: "0.0.1",
		Usage:   "swm <command>",
		Authors: []*cli.Author{
			{
				Name:  "Wael Nasreddine",
				Email: "wael.nasreddine@gmail.com",
			},
		},
		Commands: []*cli.Command{
			// server starts the server
			{
				Name:   "serve",
				Usage:  "start the gRPC server",
				Action: serve,
				Flags:  []cli.Flag{},
			},
		},
	}
}
