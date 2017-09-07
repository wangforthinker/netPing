package logCollectionTest

import (
	"testing"
	"github.com/wangforthinker/netPing/utils"
	"github.com/stretchr/testify/assert"
	"fmt"
	"time"
	"github.com/opencontainers/runc/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

func postLog(index , count int, collection *utils.LogCollection)  {
	i := 0
	for(i < count){
		i ++
		msg := fmt.Sprintf("test message %d",index + i)
		logrus.Debugf("send message:%s",msg)
		go collection.Save(msg, utils.InfoLog)
	}
}

func TestLogCollection(t *testing.T)  {
	logrus.SetLevel(logrus.DebugLevel)

	collection,err := utils.NewLogCollection("127.0.0.1:11999", "0.0.0.0")
	if(err != nil){
		assert.Nil(t, err)
		return
	}

	postLog(0, 10, collection)

	time.Sleep(time.Second * 20)

	postLog(10,10,collection)

	time.Sleep(time.Hour)
}