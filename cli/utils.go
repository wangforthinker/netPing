package cli

import (
	"net"
	"github.com/Sirupsen/logrus"
)

func get_internal() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logrus.Fatal(err.Error())
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
}
