package client

import (
	"time"
	"github.com/Sirupsen/logrus"
)


const(
	ICMP_TYPE = "icmp"
	TCP_TYPE = "tcp"
	UDP_TYPE ="udp"
)

type Options struct {
	Interval	time.Duration
	Port		int		//for tcp and udp
	PackageLength	int		//for tcp and udp
}

type PingClient interface {
	Ping() error
}

type Client struct {
	Client		PingClient
	Type		string
}

func (c *Client) Ping() error {
	logrus.Infof("start to ping %s, ", c.Type)
	return c.Client.Ping()
}
