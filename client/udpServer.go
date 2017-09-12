package client

import (
	"net"
	"fmt"
	"github.com/Sirupsen/logrus"
	"time"
)

type UdpServer struct {
	port		int
	conn		*net.UDPConn
}

type udpTempMessage struct {
	bytes		[]byte
	addr		net.Addr
}

func NewUdpServer(port int) *UdpServer {
	return &UdpServer{
		port: port,
	}
}

func (s *UdpServer) Service(ctx *context) error {
	addr,err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d",s.port))
	if(err != nil){
		return err
	}

	conn,err := net.ListenUDP("udp",addr)
	if(err != nil){
		return err
	}

	logrus.Infof("listen udp port on %d", s.port)

	s.conn = conn

	go s.runLoop(ctx)

	return nil
}

func (s *UdpServer) runLoop(ctx *context) {
	msgChan := make(chan *udpTempMessage, 100)
	go s.recvLoop(ctx, msgChan)
	go s.sendLoop(ctx, msgChan)
}

func (s *UdpServer) recvLoop(ctx *context, recv chan *udpTempMessage) {
	conn := s.conn

	for{
		select {
		case <- ctx.stop:
			logrus.Info("recevice stop chan")
			return
		default:
		}

		bytes := make([]byte, 512)
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))

		logrus.Debugf("udp server: ReadFrom Start:%s",time.Now().String())
		packetLen, ra, err := conn.ReadFrom(bytes)
		logrus.Debug("udp server: ReadFrom End:%s",time.Now().String())
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Timeout() {
					logrus.Debug("udp server: Read Timeout")
					continue
				} else {
					logrus.Debug("udp server: OpError happen", err)
					logrus.Debug("udp server: close(ctx.done)")
					logrus.Debug("udp server: wg.Done()")
					return
				}
			}
		}

		logrus.Debug("udp server: p.recv <- packet")
		logrus.Infof("udp server,recevice udp pakcet, from %s, length is %d",ra.String(), packetLen)

		select {
		case recv <- &udpTempMessage{bytes: bytes, addr: ra}:
		case <-ctx.stop:
			logrus.Debug("udp server: <-ctx.stop")
			logrus.Debug("udp server: wg.Done()")
			return
		}

	}
}

func (s *UdpServer) sendLoop(ctx *context, recv chan *udpTempMessage) {
	conn := s.conn

	for{
		select {
		case <- ctx.stop:
			logrus.Info("recevice stop chan")
			return
		case msg := <- recv:
			logrus.Debugf("send message to %s",msg.addr.String())
			_,err := conn.WriteTo(msg.bytes, msg.addr)
			if(err != nil){
				logrus.Errorf("udp server,send message to %s,error:%s",msg.addr.String(),err.Error())
			}
		}
	}
}