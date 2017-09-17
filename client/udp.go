package client

import (
	"github.com/wangforthinker/netPing/utils"
	"net"
	"fmt"
	"github.com/Sirupsen/logrus"
	"sync"
	"time"
	"errors"
	"bytes"
	"encoding/binary"
)

const(
	udpMagicType = int(0x1ff20ffe)
)

type UdpClient struct {
	servers 	[]string
	collection 	*utils.LogCollection
	opt		*Options
	//conn		*net.UDPConn
	seqMap	    	map[string]*seqSt
	connMap		map[string]*net.UDPConn
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

func (p *udpPacket) Marshal() []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, int32(p.Seq))
	binary.Write(bytesBuffer, binary.BigEndian, int32(p.Id))
	binary.Write(bytesBuffer, binary.BigEndian, int32(p.Type))
	bytes := bytesBuffer.Bytes()
	return append(bytes, p.Time...)
}

func (p *udpPacket)UnMarshal (data []byte) {
	bytesBuffer := bytes.NewBuffer(data[0:4])
	var tmp int32
	binary.Read(bytesBuffer, binary.BigEndian, &tmp)
	p.Seq =  int(tmp)

	bytesBuffer = bytes.NewBuffer(data[4:8])
	binary.Read(bytesBuffer, binary.BigEndian, &tmp)

	p.Id = int(tmp)

	bytesBuffer = bytes.NewBuffer(data[8:12])
	binary.Read(bytesBuffer, binary.BigEndian, &tmp)

	p.Type = int(tmp)
	p.Time = data[12:20]
}

type udpMessage struct {
	bytes		[]byte
	addr		net.Addr
	length		int
}

func decodeUdpPacket(bytes []byte) (*udpPacket, error) {
	//if(len(bytes) < udpPacketSize){
	//	err := fmt.Errorf("in decodeUdpPacket, recevice size is %d, not %d",len(bytes), udpPacketSize)
	//	return nil, err
	//}

	ret := &udpPacket{}
	ret.UnMarshal(bytes)
	//if(err != nil){
	//	logrus.Errorf("bytes is %s",string(bytes))
	//	return nil,err
	//}

	if(ret.Type != udpMagicType){
		err := errors.New("recevice udp packet is not magic type")
		return nil,err
	}

	return ret,nil
}

func defaultUdpRecvHandle(rtt time.Duration, seq int ,id int, addr net.Addr) {
	rttTime := rtt.Seconds()
	secondTime := int(rttTime)
	msecondTime := (rttTime - float64(secondTime)) * 1000

	logrus.Infof("udp ping to %s, seq is %d, id is %d, rtt is %ds %.3f ms",addr.String(), seq, id, secondTime, msecondTime)
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
	msg := fmt.Sprintf("udp ping, seq %d, id %d, addr %s, is time out %d s", seq, id, dst.String(), int(cli.recvTimeOut.Seconds()))
	logrus.Errorf(msg)
	go cli.collection.Save(msg, utils.ErrorLog)
}

func NewUdpClient(servers []string, opt* Options, collection *utils.LogCollection) *UdpClient {
	c := &UdpClient{
		servers: servers,
		opt: opt,
		recvHandle: defaultUdpRecvHandle,
		sendErrFunc: defaultUdpSendErrFunc,
		recvErrFunc: defaultUdpRecvErrFunc,
		sendSeqErrFunc: defaultUdpSendSeqErrFunc,
		recvTimeOutSeqErrFunc: defaultUdpRecvTimeOutErrFunc,
		seqMap: make(map[string]*seqSt),
		connMap: make(map[string]*net.UDPConn),
		collection: collection,
		recvTimeOut: time.Second * 3,
	}

	c.startLog()
	return c
}

func (c *UdpClient) Ping(ctx *context) error {

	for _,ip := range c.servers{
		addr,err := net.ResolveUDPAddr("udp",fmt.Sprintf("%s:%d",ip, c.opt.Port))
		if(err != nil){
			logrus.Error(err.Error())
			return err
		}

		conn ,err := c.initConn(ip, c.opt.Port)
		if(err != nil){
			logrus.Errorf("udp client init conn error:%s",err.Error())
			return err
		}

		c.seqMap[addr.String()] = &seqSt{
			mutex: &sync.Mutex{},
			addr: addr,
			readChMap: make(map[int]map[int]chan bool),
		}

		c.connMap[addr.String()] = conn
	}

	c.run(ctx)
	return nil
}

func (c *UdpClient) startLog()  {
	msg := fmt.Sprint("start to run udp client")
	logrus.Info(msg)
	go c.collection.Save(msg, utils.InfoLog)
}

func (c *UdpClient) run(ctx *context) error {
	go c.runLoop(ctx)
	return nil
}

func (c *UdpClient) runLoop(ct *context) {
	//wg := sync.WaitGroup{}

	id := 0
	seq := 0

	c.sendAllUdpPacket(ct, id, seq)

	ticker := time.NewTicker(c.opt.Interval)

	for key,conn := range c.connMap{
		st,ok := c.seqMap[key]
		if(!ok){
			continue
		}
		recvChan := make(chan *udpMessage, 100)
		go c.recvUdpPacket(ct, conn, recvChan)
		go c.recvLoop(ct, recvChan, st)
	}

	for{
		select {
		case <-ct.stop:
			logrus.Infof("get stop single in run loop")
			return
		case <- ticker.C:
			logrus.Debugf("send next udp packet ticker is coming")
			id,seq = addSeqAndId(id, seq)
			c.sendAllUdpPacket(ct, id, seq)
		}
	}
}

func (c *UdpClient)recvLoop(ct *context, recvChan chan *udpMessage, st *seqSt)  {
	for{
		select {
		case <-ct.stop:
			logrus.Debug("get stop single in run loop")
			return
		case recvMsg := <- recvChan:
			go c.procRecv(recvMsg, st)
		}
	}
}


func (c *UdpClient) initConn(source string ,port int) (*net.UDPConn, error){
	//if(source == ""){
	//	source = "0.0.0.0"
	//}
	//raddr,err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d",source, port))
	//if(err != nil){
	//	return nil,err
	//}

	return 	net.ListenUDP("udp", nil)

//	return net.DialUDP("udp", nil, raddr)
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

	logrus.Infof("start send all udp packet, seq %d, id %d",seq, id)

	for _,st := range c.seqMap{
		conn,ok := c.connMap[st.addr.String()]
		if(!ok){
			continue
		}

		st.mutex.Lock()

		_,ok = st.readChMap[id]
		if(!ok){
			st.readChMap[id] = make(map[int]chan bool)
		}

		readCh := make(chan bool, 1)

		st.readChMap[id][seq] = readCh

		st.mutex.Unlock()

		wg.Add(1)

		stopCh := make(chan bool, 1)

		go c.packetRespThread(readCh, stopCh, seq, id, st)
		go c.sendPacket(ct, conn, st.addr, id, seq, stopCh ,wg)
	}

	wg.Wait()
}

func (c *UdpClient) sendPacket(ct *context, conn *net.UDPConn, dst net.Addr, id int, seq int, stopCh chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(stopCh)

	times := timeToBytes(time.Now())

	msg := &udpPacket{
		Seq: seq,
		Id: id,
		Time: times,
		Type: udpMagicType,
	}

	_data := msg.Marshal()

	logrus.Debugf("udp client start to ping,seq is %d, id is %d, addr is %s", msg.Seq, msg.Id, dst.String())

	_,err := conn.WriteTo(_data, dst)
	if(err != nil){
		stopCh <- true
		c.sendErrFunc(err, c, dst, seq, id)
		return
	}

	logrus.Debugf("udp client end write to ping,seq is %d, id is %d, addr is %s", msg.Seq, msg.Id, dst.String())

	stopCh <- false
}

func (c *UdpClient) procRecv(recv *udpMessage, st *seqSt) {
	if(recv.addr.String() != st.addr.String()){
		logrus.Errorf("addr %s ,recevice udp message from %s",st.addr.String() ,recv.addr.String())
		return
	}

//	logrus.Infof("receive message, from %s, len is %d",st.addr.String(), recv.length)

	pkt,err := decodeUdpPacket(recv.bytes[0:recv.length])
	if(err != nil){
		logrus.Errorf(err.Error())
		return
	}

	id := pkt.Id
	seq := pkt.Seq
	rtt := time.Since(bytesToTime(pkt.Time))

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

func (c *UdpClient) recvUdpPacket(ctx *context, conn *net.UDPConn,recv chan *udpMessage)  {
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
		length, ra, err := conn.ReadFrom(bytes)
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
		case recv <- &udpMessage{bytes: bytes, addr: ra, length: length}:
		case <-ctx.stop:
			logrus.Debug("recvUdp(): <-ctx.stop")
			logrus.Debug("recvUdp(): wg.Done()")
			return
		}
	}
}