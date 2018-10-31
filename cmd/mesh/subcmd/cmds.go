package subcmd

import (
	"github.com/urfave/cli"
	"github.com/SevenPlusPlus/gomesh/pkg/log"
	"github.com/SevenPlusPlus/gomesh/pkg/chatroom"
)

var (
	CmdStart = cli.Command{
		Name: "start",
		Usage: "Start chat server",
		Flags: []cli.Flag {
			cli.StringFlag{
				Name: "bind-address, b",
				Usage: "bind address of server.",
				Value: "0.0.0.0:8080",
				EnvVar: "MESH_ADDRESS",
			},
		},
		Action: func(c *cli.Context) error {
			log.DefaultLogger.Infof("Starting chat server.\n")
			bindAddr := c.String("b")
			log.DefaultLogger.Infof("The server address is %s\n", bindAddr)
			chatServer := chatroom.NewChatRoomServer()
			chatServer.StartServer(bindAddr)
			return nil
		},
	}

	CmdCli = cli.Command{
		Name:               "cli",
		Usage:              "Start chat client",
		Flags:              []cli.Flag{
			cli.StringFlag{
				Name:        "server-address, s",
				Usage:       "chat server address",
				EnvVar:      "CHAT_SERVER_ADDR",
				Value:       "127.0.0.1:8080",
			},
			cli.StringFlag{
				Name:        "nick, n",
				Usage:       "client user nickname",
				Value:       "stranger",
			},
		},
		Action: func(c *cli.Context) error {
			log.DefaultLogger.Infof("Starting chat client\n")
			serverAddr := c.String("s")
			log.DefaultLogger.Infof("The server address is %s\n", serverAddr)
			nickName := c.String("n")
			log.DefaultLogger.Infof("The user nickname is %s\n", nickName)
			chatClient := chatroom.NewChatClient(serverAddr, nickName)
			chatClient.StartClient()

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
