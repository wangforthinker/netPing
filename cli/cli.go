package cli

import (
	"github.com/codegangsta/cli"
	"github.com/Sirupsen/logrus"
	"os"
)

func Run()  {
	app := cli.NewApp()
	app.Name = "netPing"
	app.Usage = "check network connections"
	app.Author = "allen.wq"
	app.Email = "wangforthinker@163.com"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "log-level, l",
			Value:  "info",
			EnvVar: "ACS_SG_LOG_LEVEL",
			Usage:  "Log level (options: debug, info, warn, error, fatal, panic)",
		},
		cli.StringFlag{
			Name:   "log-format, f",
			Value:  "console",
			EnvVar: "ACS_SG_LOG_FORMAT",
			Usage:  "Log Format (options: console, text, json)",
		},
	}

	app.Before = func(c *cli.Context) error {
		logrus.SetOutput(os.Stderr)

		level, err := logrus.ParseLevel(c.String("log-level"))
		if err != nil {
			logrus.Fatalf(err.Error())
		}
		logrus.SetLevel(level)

		return nil
	}

	app.Commands = []cli.Command{
		{
			Name: "start",
			Usage: "start sg icmp,udp,tcp connect test",
			Action: run,
			Flags:[]cli.Flag{
				fSwarmPoints, fTls, fTlsClientKey, fTlsClientCert, fContainerNumbers, fTimeInterval, fConnectContainerLabelValue, fLogServer, fUdpPort},
		},
		{
			Name: "server",
			Usage: "start as log server",
			Action: server,
			Flags:[]cli.Flag{
				fHost,
			},
		},
	}

	err := app.Run(os.Args)
	if(err != nil){
		logrus.Fatal(err.Error())
	}
}
