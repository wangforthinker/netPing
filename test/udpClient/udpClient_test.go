package udpClient

import (
	"testing"
	"github.com/wangforthinker/netPing/utils"
	"github.com/stretchr/testify/assert"
	"github.com/wangforthinker/netPing/client"
	"time"
)

func TestUdpClient(t *testing.T)  {
	collection,err := utils.NewLogCollection("127.0.0.1:11999", "0.0.0.0")
	if(err != nil){
		assert.Nil(t, err)
		return
	}

	ctx := client.NewContext()

	cli := client.NewUdpClient([]string{"127.0.0.1"}, &client.Options{Port:19999, PackageLength:128, Interval: time.Second}, collection)

	err = cli.Ping(ctx)
	assert.Nil(t, err)

	ctx.Wait()
}