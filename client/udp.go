package client

import "github.com/codegangsta/cli"

var(
	fDockerHost = cli.StringFlag{
		Name: "DockerHost",
		EnvVar: "DOCKERHOST",
		Usage: "docker host to connect",
	}
)