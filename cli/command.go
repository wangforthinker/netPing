package cli

import (
	"github.com/codegangsta/cli"
	"github.com/wangforthinker/netPing/utils"
	"github.com/Sirupsen/logrus"
	"github.com/wangforthinker/netPing/client"
	"time"
	"strconv"
	"fmt"
)

func heartBeat(logCol *utils.LogCollection, stop chan bool)  {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for{
		select {
		case <-ticker.C:
			logCol.Save(fmt.Sprintf("heartbeat time:%s", time.Now().String()), utils.InfoLog)
		case <-stop:
			return
		}
	}
}

func run(c *cli.Context)  {
	endpoints := c.String("SwarmPoints")

	logrus.Infof("start to test connect, endpoints:%s", endpoints)

	numbers := c.Int("connectNumbers")

	logrus.Infof("container numbers is %d",numbers)

	labelValue := c.String("containerLableValue")

	_udpPort := c.String("udpPort")

	icmpPing := c.Bool("icmpPing")

	udpPing := c.Bool("udpPing")

	udpPort,err := strconv.Atoi(_udpPort)
	if(err != nil){
		logrus.Fatal(err.Error())
	}

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

	icmpCtx := client.NewContext()
	udpCtx := client.NewContext()
	udpServerCtx := client.NewContext()

	if(icmpPing) {
		pingClient := client.NewICMPClient(servers, &client.Options{Interval: time.Millisecond * time.Duration(timeInterval)}, logCol)
		icmpCli := &client.Client{
			Client: pingClient,
			Type: client.ICMP_TYPE,
		}
		err = icmpCli.Ping(icmpCtx)
		if(err != nil){
			logrus.Fatalf(err.Error())
		}
	}

	if(udpPing){
		udpClient := client.NewUdpClient(servers, &client.Options{Interval: time.Millisecond * time.Duration(timeInterval) ,Port: udpPort}, logCol)
		udpServer := client.NewUdpServer(udpPort)

		udpCli := &client.Client{
			Client: udpClient,
			Type: client.UDP_TYPE,
		}

		err = udpServer.Service(udpServerCtx)
		if(err != nil){
			logrus.Fatal(err.Error())
		}

		err = udpCli.Ping(udpCtx)
		if(err != nil){
			logrus.Fatalf(err.Error())
		}
	}

	stopChan := make(chan bool)
	heartBeat(logCol, stopChan)

	udpServerCtx.Wait()
	icmpCtx.Wait()
	udpCtx.Wait()
	stopChan <- true
	close(stopChan)
}

func server(c *cli.Context)  {
	host := c.String("host")
	logrus.Infof("start server in %s",host)
	utils.NewServerAndRun(host)
}