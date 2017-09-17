package client

import (
	"net"
	"golang.org/x/net/icmp"
	"github.com/Sirupsen/logrus"
	"time"
	"golang.org/x/net/ipv4"
	"sync"
	"github.com/wangforthinker/netPing/utils"
	"fmt"
)

const(
	ProtocolICMP  = 1
)

type context struct {
	stop chan bool
	done chan bool
	err  error
}

type icmpPacket struct {
	bytes	[]byte
	addr	net.Addr
}

type seqSt struct {
	readChMap map[int]map[int]chan bool
	addr net.Addr
	mutex *sync.Mutex
}

type IcmpClient struct {
	servers     []string
	recvTimeOut time.Duration
	opt         *Options
	conn        *icmp.PacketConn
	seqMap	    map[string]*seqSt
	sendErrFunc func(err error, cli *IcmpClient, dst net.Addr, seq int, id int)
	recvHandle  func(rtt time.Duration, seq int ,id int, addr net.Addr)
	recvErrFunc func(err error, cli *IcmpClient)
	sendSeqErrFunc func(cli *IcmpClient, dst net.Addr, seq int, id int)
	recvTimeOutSeqErrFunc func(cli *IcmpClient, dst net.Addr, seq int, id int)
	collection *utils.LogCollection
}

func defaultRecvHandle(rtt time.Duration, seq int ,id int, addr net.Addr) {
	rttTime := rtt.Seconds()
	secondTime := int(rttTime)
	msecondTime := (rttTime - float64(secondTime)) * 1000

	logrus.Infof("ping to %s, seq is %d, id is %d, rtt is %d s %.3f ms",addr.String(), seq, id, secondTime, msecondTime)
}

func defaultSendErrFunc(err error, cli *IcmpClient, dst net.Addr, seq int, id int)  {
	msg := fmt.Sprintf("ping to %s, seq is %d, id is %d, err is %s",dst.String(), seq, id, err.Error())
	logrus.Errorf(msg)
	go cli.collection.Save(msg, utils.ErrorLog)
}

func defaultRecvErrFunc(err error, cli *IcmpClient)  {
	if neterr, ok := err.(*net.OpError); ok {
		if neterr.Timeout() {
			logrus.Debugf("recvICMP(): Read Timeout")
			return
		} else {
			msg := fmt.Sprintf("recv error:%s",err.Error())
			logrus.Errorf(msg)
			go cli.collection.Save(msg, utils.ErrorLog)
			return
		}
	}
}

func defaultSendSeqErrFunc(cli *IcmpClient, dst net.Addr, seq int, id int) {
	msg := fmt.Sprintf("ping icmp, seq %d, id %d, addr %s,is not get recv before next icmp", seq, id, dst.String())
	logrus.Errorf(msg)
	go cli.collection.Save(msg, utils.ErrorLog)
}

func defaultRecvTimeOutErrFunc(cli *IcmpClient, dst net.Addr, seq int, id int) {
	msg := fmt.Sprintf("ping icmp, seq %d, id %d, addr %s, is time out %d s", seq, id, dst.String(), int(cli.recvTimeOut.Seconds()))
	logrus.Errorf(msg)
	go cli.collection.Save(msg, utils.ErrorLog)
}

func NewICMPClient(servers []string, opt* Options, collection *utils.LogCollection) *IcmpClient {
	c := &IcmpClient{
		servers: servers,
		opt: opt,
		recvHandle: defaultRecvHandle,
		sendErrFunc: defaultSendErrFunc,
		recvErrFunc: defaultRecvErrFunc,
		sendSeqErrFunc: defaultSendSeqErrFunc,
		recvTimeOutSeqErrFunc: defaultRecvTimeOutErrFunc,
		seqMap: make(map[string]*seqSt),
		collection: collection,
		recvTimeOut: time.Second * 10,
	}

	c.startLog()
	return c
}

func (c *IcmpClient) startLog()  {
	msg := fmt.Sprint("start to run icmp client")
	logrus.Info(msg)
	go c.collection.Save(msg, utils.InfoLog)
}

func NewContext() *context {
	return &context{
		stop: make(chan bool),
		done: make(chan bool),
	}
}

func (ctx *context)Wait() {
	<- ctx.done
}

func (c *IcmpClient) Ping(ctx *context) error {
	conn,err := c.initConn("ip4:icmp", "")
	if(err != nil){
		return err
	}

	c.conn = conn
	c.run(ctx)

	return nil
}

func (c *IcmpClient) initConn(netProto, source string) (*icmp.PacketConn,error) {
	conn,err := icmp.ListenPacket(netProto, source)
	if(err != nil){
		logrus.Errorf("icmp listen conn error:%s",err.Error())
		return  nil,err
	}

	return conn,nil
}

func (c *IcmpClient) run(ctx *context) error {
	for _,ip := range c.servers{
		addr,err := net.ResolveIPAddr("",ip)
		if(err != nil){
			logrus.Error(err.Error())
			return err
		}

		c.seqMap[addr.String()] = &seqSt{
			mutex: &sync.Mutex{},
			addr: addr,
			readChMap: make(map[int]map[int] chan bool),
		}
	}

	go c.runLoop(ctx)
	return nil
}

func (c *IcmpClient) icmpRespThread(readCh chan bool, stopCh chan bool,seq int, id int, st *seqSt) {
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
				logrus.Infof("addr %s, seq %d, id %d , icmp resp thread is stop", st.addr.String(), seq, id)
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

func (c *IcmpClient)runLoop(ct *context) {
	//wg := sync.WaitGroup{}

	recvChan := make(chan *icmpPacket, 100)

	id := 0
	seq := 0

	c.sendAllIcmp(ct, id, seq)

	ticker := time.NewTicker(c.opt.Interval)

	go c.recvIcmp(ct, recvChan)

	for{
		select {
		case <-ct.stop:
			logrus.Debug("get stop single in run loop")
			return
		case <- ticker.C:
			logrus.Debugf("send all icmp ticker is coming")
			id,seq = addSeqAndId(id, seq)
			c.sendAllIcmp(ct, id, seq)
		case recvPacket := <- recvChan:
			go c.procRecv(recvPacket)
		}
	}
}

func (c *IcmpClient) procRecv(recv *icmpPacket) {
	var ipaddr *net.IPAddr
	switch adr := recv.addr.(type) {
	case *net.IPAddr:
		ipaddr = adr
	case *net.UDPAddr:
		ipaddr = &net.IPAddr{IP: adr.IP, Zone: adr.Zone}
	default:
		return
	}

	addr := ipaddr.String()
	if _, ok := c.seqMap[addr]; !ok {
		return
	}

	var bytes []byte
	var proto int
	bytes = ipv4Payload(recv.bytes)
	proto = ProtocolICMP

	var m *icmp.Message
	var err error
	if m, err = icmp.ParseMessage(proto, bytes); err != nil {
		return
	}

	if m.Type != ipv4.ICMPTypeEchoReply {
		return
	}

	seq := 0
	id := 0

	var rtt time.Duration
	switch pkt := m.Body.(type) {
	case *icmp.Echo:
		seq = pkt.Seq
		id = pkt.ID
		rtt = time.Since(bytesToTime(pkt.Data))
	default:
		return
	}

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

func (c *IcmpClient) sendAllIcmp(ct *context, id int, seq int) {
	wg := new(sync.WaitGroup)

	for _,st := range c.seqMap{
		st.mutex.Lock()

		_,ok := st.readChMap[id]
		if(!ok){
			st.readChMap[id] = make(map[int]chan bool)
		}

		readCh := make(chan bool, 1)

		st.readChMap[id][seq] = readCh

		st.mutex.Unlock()

		wg.Add(1)

		stopCh := make(chan bool, 1)

		go c.icmpRespThread(readCh, stopCh, seq, id, st)
		go c.sendIcmp(ct, st.addr, id, seq, stopCh ,wg)
	}

	wg.Wait()
}

func (c *IcmpClient) sendIcmp(ct *context, dst net.Addr,id int, seq int, stopCh chan bool, wg *sync.WaitGroup) {
	conn := c.conn
	defer wg.Done()
	defer close(stopCh)

	times := timeToBytes(time.Now())

	msg := &icmp.Message{
		Code: 0,
		Type: ipv4.ICMPTypeEcho,
		Body: &icmp.Echo{
			ID: id,
			Seq: seq,
			Data:times,
		},
	}

	data,err := msg.Marshal(nil)
	if(err != nil){
		stopCh <- true
		logrus.Errorf("encode error:%s",err.Error())
		return
	}

	_,err = conn.WriteTo(data, dst)
	if(err != nil){
		stopCh <- true
		c.sendErrFunc(err, c, dst, seq, id)
		return
	}

	stopCh <- false
}

func (c *IcmpClient)recvIcmp(ctx *context, recv chan *icmpPacket)  {
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

		logrus.Debugf("recvICMP(): ReadFrom Start:%s",time.Now().String())
		_, ra, err := conn.ReadFrom(bytes)
		logrus.Debug("recvICMP(): ReadFrom End:%s",time.Now().String())
		if err != nil {
			c. recvErrFunc(err, c)
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Timeout() {
					logrus.Debug("recvICMP(): Read Timeout")
					continue
				} else {
					logrus.Debug("recvICMP(): OpError happen", err)
					logrus.Debug("recvICMP(): close(ctx.done)")
					logrus.Debug("recvICMP(): wg.Done()")
					return
				}
			}
		}

		logrus.Debug("recvICMP(): p.recv <- packet")

		select {
		case recv <- &icmpPacket{bytes: bytes, addr: ra}:
		case <-ctx.stop:
			logrus.Debug("recvICMP(): <-ctx.stop")
			logrus.Debug("recvICMP(): wg.Done()")
			return
		}
	}
}

