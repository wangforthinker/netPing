package utils

import (
	"time"
	"github.com/Sirupsen/logrus"
)

type logSt struct {
	Msg		string	`json:"msg"`
	LogType		int	`json:"logType"`
	SourceIp	string	`json:"sourceIp"`
}

type LogCollection struct {
	cache chan *logSt
	stop chan bool
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
	logrus.Debugf("new log collection")
	logCli,err := NewLogClient(logServer)
	if(err != nil){
		return nil,err
	}

	logCol := &LogCollection{
		cache: make(chan *logSt, 500),
		stop: make(chan bool),
		logServer: logServer,
		sourceIp: sourceIp,
		logCli: logCli,
	}

	go logCol.run()

	return logCol,nil
}

func (c *LogCollection)Stop()  {
	c.stop <- true
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
			logrus.Debugf("in collection run,receive finish chan %v",f)
			if(f) {
				logrus.Debugf("in collection run,before exchange saveLog len is %d",len(saveLog))
				postLog = saveLog
				saveLog = []*logSt{}
				logrus.Debugf("in collection run,after exchange postlog len is %d",len(postLog))
				finish = f
			}
		case log := <- c.cache:
			logrus.Debugf("in collection run, get log: %s",log.Msg)
			saveLog = append(saveLog, log)
		case stop := <- c.stop:
			if(stop){
				logrus.Info("recevice stop singal")
				return
			}
		}
	}
}

func (c *LogCollection) saveToRemote(log []*logSt, finish chan bool) {
	defer func() {
		finish <- true
	}()

	logLen := 0
	if(log != nil){
		logLen = len(log)
	}

	logrus.Debugf("in log collection, saveToRemote recevice log len is %d",logLen)

	if(log == nil || len(log) == 0){
		return
	}

	logrus.Debugf("in log collection, saveToRemote recevice log len is %d",len(log))

	err := c.logCli.PostLog(log)
	if(err == nil){
		logrus.Debugf("post log to remote ok")
		return
	}

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	times := 0

	for{
		select {
		case <-ticker.C:
			err := c.logCli.PostLog(log)
			if(err == nil){
				logrus.Debugf("post log to remote ok")
				return
			}else{
				logrus.Errorf("post log to remote error:%s",err.Error())
			}

			times ++
			if(times > retryTimes){
				logrus.Errorf("retry %d times to post log failed,error:%s",times, err.Error())
				return
			}
		}
	}
}