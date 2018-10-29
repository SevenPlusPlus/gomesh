package main

import "github.com/urfave/cli"

var (
	cmdStart = cli.Command{
		Name: "start",
		Usage: "Start this app",
		Flags: []cli.Flag {
			cli.StringFlag{
				Name: "bind-address, addr",
				Usage: "bind address of server.",
				Value: "0.0.0.0:8080",
				EnvVar: "MESH_ADDRESS",
			},
		},
		Action: func(c *cli.Context) error {
			return nil
		},
	}

	cmdStop = cli.Command{
		Name: "stop",
		Usage: "Stop this app",
		Action: func(c *cli.Context) error {
			return nil
		},
	}
)
