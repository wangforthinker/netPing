package client

import (
	"github.com/wangforthinker/netPing/utils"
	"net"
	"fmt"
	"github.com/Sirupsen/logrus"
	"sync"
	"time"
	"encoding/json"
	"errors"
)

const(
	udpMagicType = int(0x1ff20ffe)
	udpPacketSize = 20
)

type UdpClient struct {
	servers 	[]string
	collection 	*utils.LogCollection
	opt		*Options
	conn		*net.UDPConn
	seqMap	    	map[string]*seqSt
	recvTimeOut	time.Duration
	sendErrFunc 	func(err error, cli *UdpClient, dst net.Addr, seq int, id int)
	recvHandle  	func(rtt time.Duration, seq int ,id int, addr net.Addr)
	recvErrFunc 	func(err error, cli *UdpClient)
	sendSeqErrFunc 	func(cli *UdpClient, dst net.Addr, seq int, id int)
	recvTimeOutSeqErrFunc func(cli *UdpClient, dst net.Addr, seq int, id int)
}

type udpPacket struct {
	Seq		int	`json:"seq"`
	Id		int	`json:"int"`
	Type		int	`json:"type"`
	Time		[]byte	`json:"time"`
}

type udpMessage struct {
	bytes		[]byte
	addr		net.Addr
}

func decodeUdpPacket(bytes []byte) (*udpPacket, error) {
	if(len(bytes) < udpPacketSize){
		err := fmt.Errorf("in decodeUdpPacket, recevice size is %d, not %d",len(bytes), udpPacketSize)
		return nil, err
	}

	ret := &udpPacket{}
	err := json.Unmarshal(bytes, ret)
	if(err != nil){
		return nil,err
	}

	if(ret.Type != udpMagicType){
		err = errors.New("recevice udp packet is not magic type")
		return nil,err
	}

	return ret,nil
}

func defaultUdpRecvHandle(rtt time.Duration, seq int ,id int, addr net.Addr) {
	rttTime := rtt.Seconds()
	secondTime := int(rttTime)
	msecondTime := int((rttTime - float64(secondTime)) * 1000)

	logrus.Infof("udp to %s, seq is %d, id is %d, rtt is %ds %dms",addr.String(), seq, id, secondTime, msecondTime)
}

func defaultUdpSendErrFunc(err error, cli *UdpClient, dst net.Addr, seq int, id int)  {
	msg := fmt.Sprintf("udp to %s, seq is %d, id is %d, err is %s",dst.String(), seq, id, err.Error())
	logrus.Errorf(msg)
	go cli.collection.Save(msg, utils.ErrorLog)
}

func defaultUdpRecvErrFunc(err error, cli *UdpClient)  {
	if neterr, ok := err.(*net.OpError); ok {
		if neterr.Timeout() {
			logrus.Debugf("recvudp(): Read Timeout")
			return
		} else {
			msg := fmt.Sprintf("recv error:%s",err.Error())
			logrus.Errorf(msg)
			go cli.collection.Save(msg, utils.ErrorLog)
			return
		}
	}
}

func defaultUdpSendSeqErrFunc(cli *UdpClient, dst net.Addr, seq int, id int) {
	msg := fmt.Sprintf("udp ping, seq %d, id %d, addr %s,is not get recv before next icmp", seq, id, dst.String())
	logrus.Errorf(msg)
	go cli.collection.Save(msg, utils.ErrorLog)
}

func defaultUdpRecvTimeOutErrFunc(cli *UdpClient, dst net.Addr, seq int, id int) {
	msg := fmt.Sprintf("udp ping, seq %d, id %d, addr %s, is time out %d s", seq, id, dst.String(), cli.recvTimeOut)
	logrus.Errorf(msg)
	go cli.collection.Save(msg, utils.ErrorLog)
}

func NewUcpClient(servers []string, opt* Options, collection *utils.LogCollection) *UdpClient {
	c := &UdpClient{
		servers: servers,
		opt: opt,
		recvHandle: defaultUdpRecvHandle,
		sendErrFunc: defaultUdpSendErrFunc,
		recvErrFunc: defaultUdpRecvErrFunc,
		sendSeqErrFunc: defaultUdpSendSeqErrFunc,
		recvTimeOutSeqErrFunc: defaultUdpRecvTimeOutErrFunc,
		seqMap: make(map[string]*seqSt),
		collection: collection,
		recvTimeOut: time.Second * 10,
	}

	c.startLog()
	return c
}

func (c *UdpClient) Ping() error {
	conn,err := c.initConn("", c.opt.Port)
	if(err != nil){
		return err
	}

	c.conn = conn
	c.run()

	return nil
}

func (c *UdpClient) startLog()  {
	msg := fmt.Sprint("start to run udp client")
	logrus.Info(msg)
	go c.collection.Save(msg, utils.InfoLog)
}

func (c *UdpClient) run() error {
	ct := newContext()
	for _,ip := range c.servers{
		addr,err := net.ResolveUDPAddr("udp4",fmt.Sprintf("%s:%d",ip, c.opt.Port))
		if(err != nil){
			logrus.Error(err.Error())
			return err
		}

		serverAddr := &net.IPAddr{IP:net.ParseIP(ip)}

		c.seqMap[serverAddr.String()] = &seqSt{
			mutex: &sync.Mutex{},
			addr: addr,
			readChMap: make(map[int]map[int]chan bool),
		}
	}

	c.runLoop(ct)
	return nil
}

func (c *UdpClient) runLoop(ct *context) {
	//wg := sync.WaitGroup{}

	recvChan := make(chan *udpMessage, 100)

	id := 0
	seq := 0

	c.sendAllUdpPacket(ct, id, seq)

	ticker := time.NewTicker(c.opt.Interval)

	go c.recvUdpPacket(ct, recvChan)

	for{
		select {
		case <-ct.stop:
			logrus.Debug("get stop single in run loop")
			return
		case <- ticker.C:
			logrus.Debugf("send next udp packet ticker is coming")
			id,seq = addSeqAndId(id, seq)
			c.sendAllUdpPacket(ct, id, seq)
		case recvMsg := <- recvChan:
			go c.procRecv(recvMsg)
		}
	}
}


func (c *UdpClient) initConn(source string, port int) (*net.UDPConn, error){
	if(source == ""){
		source = "0.0.0.0"
	}
	addr,err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d",source, port))
	if(err != nil){
		return nil,err
	}

	return net.ListenUDP("udp4", addr)
}

func (c *UdpClient) packetRespThread(readCh chan bool, stopCh chan bool,seq int, id int, st *seqSt) {
	ticker := time.NewTicker(c.recvTimeOut)
	defer ticker.Stop()

	for {
		out := false
		select {
		case <-ticker.C:
			c.recvTimeOutSeqErrFunc(c, st.addr, seq, id)
			out = true
			break
		case <-readCh:
			out = true
			break
		case stop := <-stopCh:
			if(stop) {
				logrus.Infof("addr %s, seq %d, id %d , udp resp thread is stop", st.addr.String(), seq, id)
				out = true
				break
			}
		}

		if(out){
			break
		}
	}

	st.mutex.Lock()
	defer st.mutex.Unlock()

	close(readCh)
	delete(st.readChMap[id], seq)
}

func (c *UdpClient) sendAllUdpPacket(ct *context, id int, seq int) {
	wg := new(sync.WaitGroup)

	for _,st := range c.seqMap{
		st.mutex.Lock()

		_,ok := st.readChMap[id]
		if(!ok){
			st.readChMap[id] = make(map[int]chan bool)
		}

		readCh := make(chan bool)

		st.readChMap[id][seq] = readCh

		st.mutex.Unlock()

		wg.Add(1)

		stopCh := make(chan bool)

		go c.packetRespThread(readCh, stopCh, seq, id, st)
		go c.sendPacket(ct, st.addr, id, seq, stopCh ,wg)
	}

	wg.Wait()
}

func (c *UdpClient) sendPacket(ct *context, dst net.Addr, id int, seq int, stopCh chan bool, wg *sync.WaitGroup) {
	conn := c.conn
	defer wg.Done()
	defer close(stopCh)

	times := timeToBytes(time.Now())

	msg := &udpPacket{
		Seq: seq,
		Id: id,
		Time: times,
		Type: udpMagicType,
	}

	_data,err := json.Marshal(msg)
	if(err != nil){
		stopCh <- true
		logrus.Errorf("encode error:%s",err.Error())
		return
	}

	data := packetSuppleToSize(_data, c.opt.PackageLength)

	_,err = conn.WriteTo(data, dst)
	if(err != nil){
		stopCh <- true
		c.sendErrFunc(err, c, dst, seq, id)
	}

	stopCh <- false
}

func (c *UdpClient) procRecv(recv *udpMessage) {
	var ipaddr *net.IPAddr
	switch adr := recv.addr.(type) {
	case *net.UDPAddr:
		ipaddr = &net.IPAddr{IP: adr.IP, Zone: adr.Zone}
	default:
		return
	}

	addr := ipaddr.String()
	if _, ok := c.seqMap[addr]; !ok {
		return
	}

	pkt,err := decodeUdpPacket(recv.bytes[:udpPacketSize])
	if(err != nil){
		logrus.Errorf(err.Error())
		return
	}

	id := pkt.Id
	seq := pkt.Seq
	rtt := time.Since(bytesToTime(pkt.Time))

	if st, ok := c.seqMap[addr]; ok {
		c.recvHandle(rtt, seq, id, st.addr)

		st.mutex.Lock()
		defer st.mutex.Unlock()

		_,ok := st.readChMap[id]
		if(!ok){
			return
		}

		_,ok = st.readChMap[id][seq]
		if(!ok){
			return
		}

		st.readChMap[id][seq] <- true
	}
}

func (c *UdpClient) recvUdpPacket(ctx *context, recv chan *udpMessage)  {
	conn := c.conn

	for {
		select {
		case <-ctx.stop:
			logrus.Debug("stop recv icmp")
			return
		default:
		}

		bytes := make([]byte, 512)
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))

		logrus.Debugf("recvUdp(): ReadFrom Start:%s",time.Now().String())
		_, ra, err := conn.ReadFrom(bytes)
		logrus.Debug("recvICMP(): ReadFrom End:%s",time.Now().String())
		if err != nil {
			c. recvErrFunc(err, c)
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Timeout() {
					logrus.Debug("recvUdp(): Read Timeout")
					continue
				} else {
					logrus.Debug("recvUdp(): OpError happen", err)
					logrus.Debug("recvUdp(): close(ctx.done)")
					logrus.Debug("recvUdp(): wg.Done()")
					return
				}
			}
		}

		logrus.Debug("recvUdp(): p.recv <- packet")

		select {
		case recv <- &udpMessage{bytes: bytes, addr: ra}:
		case <-ctx.stop:
			logrus.Debug("recvUdp(): <-ctx.stop")
			logrus.Debug("recvUdp(): wg.Done()")
			return
		}
	}
}