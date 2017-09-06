package cli

import (
	"github.com/codegangsta/cli"
	"github.com/wangforthinker/netPing/utils"
	"github.com/Sirupsen/logrus"
	"github.com/wangforthinker/netPing/client"
	"time"
)

func run(c *cli.Context)  {
	endpoints := c.String("SwarmPoints")

	logrus.Infof("start to test connect, endpoints:%s", endpoints)

	numbers := c.Int("connectNumbers")

	logrus.Infof("container numbers is %d",numbers)

	labelValue := c.String("containerLableValue")

	dockerClient,err := utils.NewDockerClient(endpoints, nil)
	if(err != nil){
		logrus.Fatal(err.Error())
	}

	servers,err := utils.GetServersIPs(dockerClient, numbers, labelValue)
	if(err != nil){
		logrus.Fatal(err.Error())
	}

	timeInterval := c.Int("timeInterval")
	logrus.Infof("timeInterval is %d ms", timeInterval)

	localIp := get_internal()
	logServerAddr := c.String("logServer")

	logrus.Infof("local ip is %s, log server addr is %s", localIp, logServerAddr)

	logCol,err := utils.NewLogCollection(logServerAddr, localIp)
	if(err != nil){
		logrus.Fatal(err.Error())
	}

	pingClient := client.NewICMPClient(servers, &client.Options{Interval: time.Millisecond * time.Duration(timeInterval)}, logCol)

	cli := &client.Client{
		Client: pingClient,
		Type: client.ICMP_TYPE,
	}

	err = cli.Ping()
	if(err != nil){
		logrus.Fatalf(err.Error())
	}
}

func server(c *cli.Context)  {
	host := c.String("host")
	logrus.Infof("start server in %s",host)
	utils.NewServerAndRun(host)
}