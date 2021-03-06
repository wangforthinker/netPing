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

	fLogServer = cli.StringFlag{
		Name: "logServer",
		Usage:"log server addr",
		EnvVar: env.LOG_SERVER_ADDR,
	}

	fHost = cli.StringFlag{
		Name: "host",
		Usage:"server host,(0.0.0.0:11999)",
		EnvVar: env.SERVER_HOST,
		Value: "0.0.0.0:11999",
	}

	fUdpPort = cli.StringFlag{
		Name: "udpPort",
		Usage:"udp listen port",
		EnvVar: env.UDP_LISTEN_PORT,
		Value:"19999",
	}

	fIcmpPing = cli.BoolFlag{
		Name: "icmpPing",
		Usage: "open icmp ping",
		EnvVar: env.ICMP_PING,
	}

	fUdpPing = cli.BoolFlag{
		Name: "udpPing",
		Usage: "open udp ping",
		EnvVar: env.UDP_PING,
	}
)