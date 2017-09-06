package utils

import (
	"net/http"
	"net/url"
	"fmt"
	"github.com/Sirupsen/logrus"
	"encoding/json"
	"bytes"
)

const(
	savePath = "saveLog"
)

type LogClient struct {
	server	 string
	url	*url.URL
}

func NewLogClient(server string) (*LogClient,error) {
	ul,err := url.Parse(fmt.Sprintf("http://%s/%s",server, savePath))
	if(err != nil){
		return nil,err
	}

	return &LogClient{server:server, url:ul},nil
}

func (cli *LogClient)PostLog(logs []*logSt) error {
	data,err := json.Marshal(logs)
	if(err != nil){
		return err
	}

	req,err := http.NewRequest(http.MethodPost, cli.url.String(), bytes.NewReader(data))
	if(err != nil){
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp,err := http.DefaultClient.Do(req)
	if(err != nil){
		logrus.Errorf("post log error:%s", err.Error())
		return err
	}

	defer resp.Body.Close()
	if(resp.StatusCode >= 400){
		return fmt.Errorf("resp code is %d",resp.StatusCode)
	}

	return nil
}
