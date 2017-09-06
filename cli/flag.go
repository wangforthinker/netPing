package cli

import (
	"github.com/codegangsta/cli"
	"github.com/wangforthinker/netPing/env"
)

var(
	fSwarmPoints = cli.StringFlag{
		Name: "SwarmPoints",
		Usage: "swarm points to connect",
		EnvVar: env.SWARM_ENDPOINTS,
	}

	fTls = cli.BoolFlag{
		Name: "tls",
		Usage: "tls enable",
		EnvVar: env.TLS,
	}

	fTlsClientKey = cli.StringFlag{
		Name: "tlskey",
		Usage:"tls key file path",
		EnvVar: env.TLS_CLIENT_KEY,
	}

	fTlsClientCert = cli.StringFlag{
		Name: "tlscert",
		Usage: "tls cert file path",
		EnvVar: env.TLS_CLIENT_CERT,
	}

	fContainerNumbers = cli.IntFlag{
		Name: "connectNumbers",
		Usage: "numbers of containers to connect",
		Value: 0,
		EnvVar: env.CONNECT_CONTAINER_NUMBER,
	}

	fTimeInterval = cli.IntFlag{
		Name: "timeInterval",
		Usage: "time interval ms",
		Value: 1000,
		EnvVar: env.TIME_INTERVAL_MS,
	}

	fConnectContainerLabelValue = cli.StringFlag{
		Name: "containerLableValue",
		Usage:"label value to connect",
		EnvVar: env.CONNECT_CONTAINER_LABEL_VALUE,
	}
)