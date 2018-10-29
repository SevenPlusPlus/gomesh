package subcmd

import (
	"github.com/urfave/cli"
	"github.com/SevenPlusPlus/gomesh/pkg/log"
)

var (
	CmdStart = cli.Command{
		Name: "start",
		Usage: "Start this app",
		Flags: []cli.Flag {
			cli.StringFlag{
				Name: "bind-address, b",
				Usage: "bind address of server.",
				Value: "0.0.0.0:8080",
				EnvVar: "MESH_ADDRESS",
			},
		},
		Action: func(c *cli.Context) error {
			log.DefaultLogger.Infof("Starting the server.")
			bindAddr := c.String("b")
			log.DefaultLogger.Infof("The server address is %s\n", bindAddr)
			return nil
		},
	}

	CmdStop = cli.Command{
		Name: "stop",
		Usage: "Stop this app",
		Action: func(c *cli.Context) error {
			return nil
		},
	}
)
