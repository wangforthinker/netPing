package client

import (
	"net"
	"golang.org/x/net/icmp"
	"github.com/Sirupsen/logrus"
	"time"
	"golang.org/x/net/ipv4"
	"sync"
)

const(
	ProtocolICMP  = 1
)

type context struct {
	stop chan bool
	done chan bool
	err  error
}

type packet struct {
	bytes	[]byte
	addr	net.Addr
}

type seqSt struct {
	seq int
	id int
	read bool
	addr net.Addr
}

type IcmpClient struct {
	servers     []string
	opt         *Options
	conn        *icmp.PacketConn
	addrMap     map[string]net.Addr
	seqMap	    map[string]*seqSt
	seqMapMutex *sync.Mutex
	sendErrFunc func(err error, cli *IcmpClient, dst net.Addr, seq int, id int)
	recvHandle  func(rtt time.Duration, seq int ,id int, addr net.Addr)
	recvErrFunc func(err error, cli *IcmpClient)
	sendSeqErrFunc func(cli *IcmpClient, dst net.Addr, seq int, id int)
}

func timeToBytes(t time.Time) []byte {
	nsec := t.UnixNano()
	b := make([]byte, 8)
	for i := uint8(0); i < 8; i++ {
		b[i] = byte((nsec >> ((7 - i) * 8)) & 0xff)
	}
	return b
}

func bytesToTime(b []byte) time.Time {
	var nsec int64
	for i := uint8(0); i < 8; i++ {
		nsec += int64(b[i]) << ((7 - i) * 8)
	}
	return time.Unix(nsec/1000000000, nsec%1000000000)
}

func defaultRecvHandle(rtt time.Duration, seq int ,id int, addr net.Addr) {
	rttTime := rtt.Seconds()
	secondTime := int(rttTime)
	msecondTime := int((rttTime - float64(secondTime)) * 1000)

	logrus.Infof("ping to %s, seq is %d, id is %d, rtt is %ds %dms",addr.String(), seq, id, secondTime, msecondTime)
}

func defaultSendErrFunc(err error, cli *IcmpClient, dst net.Addr, seq int, id int)  {
	logrus.Errorf("ping to %s, seq is %d, id is %d, err is %s",dst.String(), seq, id, err.Error())
}

func defaultRecvErrFunc(err error, cli *IcmpClient)  {
	if neterr, ok := err.(*net.OpError); ok {
		if neterr.Timeout() {
			logrus.Debugf("recvICMP(): Read Timeout")
			return
		} else {
			logrus.Errorf("recv error:%s",err.Error())
			return
		}
	}
}

func defaultSendSeqErrFunc(cli *IcmpClient, dst net.Addr, seq int, id int) {
	logrus.Errorf("ping icmp, seq %d, id %d, addr %s,is not get recv before next icmp", seq, id, dst.String())
}

func NewICMPClient(servers []string, opt* Options) *IcmpClient {
	return &IcmpClient{
		servers: servers,
		opt: opt,
		recvHandle: defaultRecvHandle,
		sendErrFunc: defaultSendErrFunc,
		recvErrFunc: defaultRecvErrFunc,
		sendSeqErrFunc: defaultSendSeqErrFunc,
		addrMap: make(map[string]net.Addr),
		seqMap: make(map[string]*seqSt),
		seqMapMutex: new(sync.Mutex),
	}
}

func newContext() *context {
	return &context{
		stop: make(chan bool),
		done: make(chan bool),
	}
}

func ipv4Payload(b []byte) []byte {
	if len(b) < ipv4.HeaderLen {
		return b
	}
	hdrlen := int(b[0]&0x0f) << 2
	return b[hdrlen:]
}

func (c *IcmpClient) Ping() error {
	conn,err := c.initConn("ip4:icmp", "")
	if(err != nil){
		return err
	}

	c.conn = conn
	c.run()

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

func (c *IcmpClient) run() error {
	ct := newContext()
	for _,ip := range c.servers{
		addr,err := net.ResolveIPAddr("",ip)
		if(err != nil){
			logrus.Error(err.Error())
			return err
		}

		c.addrMap[addr.String()] = addr
		c.seqMap[addr.String()] = &seqSt{id:0,seq:0,addr:addr,read:true}
	}

	c.runLoop(ct)
	return nil
}

func addSeqAndId(id , seq int) (int, int) {
	add := 0

	if(seq == 0xff){
		seq = 0
		add = 1
	}else{
		seq ++
	}

	if(add ==1){
		if(id == 0xff){
			id = 0
		}else{
			id ++
		}
	}

	return id,seq
}

func (c *IcmpClient)runLoop(ct *context) {
	//wg := sync.WaitGroup{}

	recvChan := make(chan *packet, 1)

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
			c.procRecv(recvPacket, c.addrMap, id, seq)
		}
	}
}

func (c *IcmpClient) procRecv(recv *packet, queue map[string]net.Addr, id int ,seq int) {
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
	if _, ok := c.addrMap[addr]; !ok {
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

	var rtt time.Duration
	switch pkt := m.Body.(type) {
	case *icmp.Echo:
		if pkt.ID == id && pkt.Seq == seq {
			rtt = time.Since(bytesToTime(pkt.Data))
		}
	default:
		return
	}

	if netAddr, ok := queue[addr]; ok {
		c.recvHandle(rtt, seq, id, netAddr)
	}

	c.seqMapMutex.Lock()

	if st,ok := c.seqMap[addr];ok{
		if(st.seq == seq && st.id == id){
			st.read = true
		}
	}

	c.seqMapMutex.Unlock()
}

func (c *IcmpClient) sendAllIcmp(ct *context, id int, seq int) {
	wg := new(sync.WaitGroup)

	c.seqMapMutex.Lock()
	for _,st := range c.seqMap{
		if(!st.read){
			c.sendSeqErrFunc(c, st.addr, st.seq, st.id)
		}

		st.read = false
		st.seq = seq
		st.id = id
	}
	c.seqMapMutex.Unlock()

	for _,addr := range c.addrMap{
		wg.Add(1)
		go c.sendIcmp(ct, addr, id, seq, wg)
	}

	wg.Wait()
}

func (c *IcmpClient) sendIcmp(ct *context, dst net.Addr,id int, seq int, wg *sync.WaitGroup) error {
	conn := c.conn
	defer wg.Done()

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
		logrus.Errorf("encode error:%s",err.Error())
		return err
	}

	_,err = conn.WriteTo(data, dst)
	if(err != nil){
		c.sendErrFunc(err, c, dst, seq, id)
	}

	return err
}

func (c *IcmpClient)recvIcmp(ctx *context, recv chan *packet)  {
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
		case recv <- &packet{bytes: bytes, addr: ra}:
		case <-ctx.stop:
			logrus.Debug("recvICMP(): <-ctx.stop")
			logrus.Debug("recvICMP(): wg.Done()")
			return
		}
	}
}

