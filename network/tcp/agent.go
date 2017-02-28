package tcp

import (
	"bufio"
	"github.com/sydnash/lotou/core"
	"github.com/sydnash/lotou/log"
	"net"
	"sync"
	"time"
)

/*
	recieve message from network
	split message into package
	send package to dest service

	reciev other inner message and process(such as write to network; close; change dest service)

	agent has tow goroutine:
		one to read message from tcpConnector and send to dest
		one read inner message chan and process

	this also has a timeout for first data arrived after agent create.
*/

type Agent struct {
	*core.Skeleton
	Con                  *net.TCPConn
	Dest                 uint
	hasDataArrived       bool
	leftTimeBeforArrived int
	inbuffer             *bufio.Reader
	outbuffer            *bufio.Writer
	bufferMutxt          sync.Mutex
	closeChan            chan byte
}

const (
	AGENT_CLOSED = iota
	AGENT_DATA
	AGENT_ARRIVE
)
const (
	AGENT_CMD_SEND = iota
)

func (a *Agent) OnInit() {
	a.hasDataArrived = false
	a.leftTimeBeforArrived = 5000
	a.inbuffer = bufio.NewReader(a.Con)
	a.outbuffer = bufio.NewWriter(a.Con)
	a.closeChan = make(chan byte)
	go func() {
	EXIT:
		for {
			select {
			case <-a.closeChan:
				break EXIT
			default:
				pack, err := Subpackage(self.inbuffer)
				if err != nil {
					log.Error("agent read msg failed: %s", err)
					self.onConnectError()
					break
				}
				if self.timeout != nil {
					self.timeout.Stop()
					self.timeout = nil
				}
				a.RawSend(a.Dest, core.MSG_TYPE_SOKECT, AGENT_DATA, pack)
			}
		}
	}()
}

func NewAgent(con *net.TCPConn, dest uint) *Agent {
	a := &Agent{Con: con, Dest: dest, Base: core.NewSkeleton()}
	return a
}

func (a *Agent) OnMainLoop(dt int) {
	if !a.hasDataArrived {
		a.leftTimeBeforArrived -= dt
		if a.leftTimeBeforArrived < 0 {
			a.close()
		}
	}
}

func (a *Agent) OnNormalMSG(src uint, data ...interface{}) {
	cmd := data[0].(int)
	if cmd == AGENT_CMD_SEND {
		a.Con.SetWriteDeadline(time.Now().Add(time.Second * 20))
		msg := data[1].([]byte)
		if _, err := a.outbuffer.Write(msg); err != nil {
			log.Error("agent write msg failed: %s", err)
			self.onConnectError()
		}
		err = a.outbuffer.Flush()
		if err != nil {
			log.Error("agent write msg failed: %s", err)
			self.onConnectError()
		}
	}
}

func (a *Agent) OnDestroy() {
	a.close()
}

func (self *Agent) onConnectError() {
	self.RawSend(self.Dest, core.MSG_TYPE_NORMAL, AGENT_CLOSED)
	self.SendClose(self.Id)
}
func (self *Agent) close() {
	log.Info("close agent. %v", self.Con.RemoteAddr())
	self.Con.Close()
	close(self.closeChan)
}
