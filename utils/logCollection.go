package utils

import (
	"time"
	"github.com/opencontainers/runc/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

type logSt struct {
	Msg		string	`json:"msg"`
	LogType		int	`json:"logType"`
	SourceIp	string	`json:"sourceIp"`
}

type LogCollection struct {
	cache chan *logSt
	logServer string
	sourceIp  string

	logCli	*LogClient
}

const(
	retryTimes = 100

	ErrorLog = 1
	InfoLog = 2
	WarnLog = 3
)

func NewLogCollection(logServer string, sourceIp string) (*LogCollection,error) {
	logCli,err := NewLogClient(logServer)
	if(err != nil){
		return nil,err
	}

	logCol := &LogCollection{
		cache: make(chan *logSt, 500),
		logServer: logServer,
		sourceIp: sourceIp,
		logCli: logCli,
	}

	go logCol.run()

	return logCol,nil
}

func (c *LogCollection)Save(msg string, logType int) error {
	st := &logSt{
		Msg: msg,
		LogType: logType,
		SourceIp: c.sourceIp,
	}

	c.cache <- st
	return nil
}

func (c *LogCollection) run() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	postLog := []*logSt{}
	saveLog := []*logSt{}
	finish := true

	finishChan := make(chan bool, 1)

	for{
		select {
		case <- ticker.C:
			if(finish){
				finish = false
				go c.saveToRemote(postLog, finishChan)
			}
		case f := <- finishChan:
			if(f) {
				postLog = saveLog
				saveLog = []*logSt{}
				finish = f
			}
		case log := <- c.cache:
			saveLog = append(saveLog, log)
		}
	}
}

func (c *LogCollection) saveToRemote(log []*logSt, finish chan bool) {
	err := c.logCli.PostLog(log)
	if(err == nil){
		return
	}

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	defer func() {
		finish <- true
	}()

	times := 0

	for{
		select {
		case <-ticker.C:
			err := c.logCli.PostLog(log)
			if(err == nil){
				return
			}

			times ++
			if(times > retryTimes){
				logrus.Errorf("retry %d times to post log failed,error:%s",times, err.Error())
				return
			}
		}
	}
}