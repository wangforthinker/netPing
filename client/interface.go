package client

import (
	"time"
	"github.com/Sirupsen/logrus"
	"golang.org/x/net/ipv4"
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


func addSeqAndId(id , seq int) (int, int) {
	add := 0

	if(seq == 0xffff){
		seq = 0
		add = 1
	}else{
		seq ++
	}

	if(add ==1){
		if(id == 0xffff){
			id = 0
		}else{
			id ++
		}
	}

	return id,seq
}

func ipv4Payload(b []byte) []byte {
	if len(b) < ipv4.HeaderLen {
		return b
	}
	hdrlen := int(b[0]&0x0f) << 2
	return b[hdrlen:]
}

func timeToBytes(t time.Time) []byte {
	nsec := t.UnixNano()
	b := make([]byte, 8)
	for i := uint8(0); i < 8; i++ {
		b[i] = byte((nsec >> ((7 - i) * 8)) & 0xff)
	}
	return b
}

func bytesToTime(b []byte) time.Time {
	var nsec int64
	for i := uint8(0); i < 8; i++ {
		nsec += int64(b[i]) << ((7 - i) * 8)
	}
	return time.Unix(nsec/1000000000, nsec%1000000000)
}

func packetSuppleToSize(cur []byte, size int) []byte {
	if(len(cur) >= size){
		return cur
	}

	ret := make([]byte, size, 'x')
	copy(ret[:len(cur)], cur)

	return ret
}