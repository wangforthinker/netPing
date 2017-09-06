package utils

import (
	"crypto/tls"
	"github.com/samalba/dockerclient"
	"fmt"
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"time"
)

const(
	SGLabelKey = "com.alipay.acs.sg.test.container"
)

func NewDockerClient(endpoints string,tls *tls.Config) (*dockerclient.DockerClient, error) {
	return dockerclient.NewDockerClient(endpoints, tls)
}

func getContainers(client *dockerclient.DockerClient, labelValue string, labelKey string) ([]dockerclient.Container, error) {
	filter := map[string]map[string]bool{}

	filter["label"] = map[string]bool{
		fmt.Sprintf("%s=%s",labelKey, labelValue):true,
	}

	data,err := json.Marshal(filter)
	if(err != nil){
		return nil,err
	}

	return client.ListContainers(true, false, string(data))
}

func getContainerIp(cnt *dockerclient.Container) (string,error) {
	for _,network := range cnt.NetworkSettings.Networks{
		if(network.IPAddress != ""){
			return network.IPAddress,nil
		}
	}

	return "",fmt.Errorf("not found container %s network", cnt.Names[0])
}

func GetServersIPs(client *dockerclient.DockerClient, numbers int, labelValue string) ([]string,error) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			containers, err := getContainers(client, labelValue, SGLabelKey)
			if (err != nil) {
				return nil, err
			}

			if (len(containers) < numbers) {
				logrus.Infof("containers number is %d,not %d", len(containers), numbers)
			}else{
				ret := make([]string, len(containers))
				for i,cnt := range containers{
					ip,err := getContainerIp(&cnt)
					if(err != nil){
						return nil,err
					}
					logrus.Infof("server ip is %s",ip)
					ret[i] = ip
				}

				return ret,nil
			}
		}
	}
}
