package main

import (
	"github.com/urfave/cli"
	"os"
	"time"
	"github.com/SevenPlusPlus/gomesh/pkg/log"
	"github.com/SevenPlusPlus/gomesh/cmd/mesh/subcmd"
)

var (
	Name = "mesh"
	Version = "1.0.0"
)

func main() {
	app := cli.NewApp()
	app.Name = Name
	app.Version = Version
	app.Author = "wqj"
	app.Compiled = time.Now()
	app.Copyright = "(c) 2018 Qijia.Wang"
	app.Usage = "A simple archetype used to build a network related app."

	app.Commands = []cli.Command{
		subcmd.CmdStart,
		subcmd.CmdStop,
	}

	app.Action = func(c *cli.Context)error {
		cli.ShowAppHelp(c)
		log.InitDefaultLogger("stdout", log.DEBUG)
		c.App.Setup()
		return nil
	}

	app.Run(os.Args)
}
