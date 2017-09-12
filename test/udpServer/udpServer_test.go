package udpServer

import (
	"testing"
	"github.com/wangforthinker/netPing/client"
	"github.com/stretchr/testify/assert"
)

func TestUdpServer(t *testing.T)  {
	ctx := client.NewContext()
	server := client.NewUdpServer(19999)
	err := server.Service(ctx)
	assert.Nil(t, err)

	ctx.Wait()
}
