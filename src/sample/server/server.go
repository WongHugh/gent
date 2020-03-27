package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	. "gent/src/codec"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	"log"
	"sync/atomic"
	"time"
)

type codecServer struct {
	*gnet.EventServer
	addr       string
	multicore  bool
	async      bool
	codec      gnet.ICodec
	workerPool *goroutine.Pool
	connected    int32
}

func (cs *codecServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("Test codec server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
	return
}

func (cs *codecServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	if cs.async {
		data := append([]byte{}, frame...)
		_ = cs.workerPool.Submit(func() {
			c.AsyncWrite(data)
		})
		return
	}
	out = frame
	fmt.Printf("out %02x\n", out)
	return
}
func (cs *codecServer) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	c.SetContext(c)
	atomic.AddInt32(&cs.connected, 1)
	out = []byte(c.RemoteAddr().String())
	if c.LocalAddr() == nil {
		panic("nil local addr")
	}
	if c.RemoteAddr() == nil {
		panic("nil local addr")
	}
	return
	return
}



func testCodecServe(addr string, multicore, async bool, codec gnet.ICodec) {
	var err error
	if codec == nil {
		encoderConfig := EncoderConfig{
			ByteOrder:                       binary.BigEndian,
			LengthFieldLength:               1,
			LengthAdjustment:                0,
			LengthIncludesLengthFieldLength: false,
			Header:                          []byte{0x55, 0xaa},
			AddCheckData:                    true,
		}
		decoderConfig := DecoderConfig{
			ByteOrder:           binary.BigEndian,
			LengthFieldOffset:   2,
			LengthFieldLength:   1,
			LengthAdjustment:    1,
			InitialBytesToStrip: 3,
			FinalBytesToStrip:   1,
			CheckData:true,
		}
		codec = NewLengthFieldBasedFrameCodec(encoderConfig, decoderConfig)
	}
	cs := &codecServer{addr: addr, multicore: multicore, async: async, codec: codec, workerPool: goroutine.Default()}
	err = gnet.Serve(cs, addr, gnet.WithMulticore(multicore), gnet.WithTCPKeepAlive(time.Minute*5), gnet.WithCodec(codec))
	if err != nil {
		panic(err)
	}
}

func main() {
	var port int
	var multicore bool

	// Example command: go run server.go --port 9000 --multicore=true
	flag.IntVar(&port, "port", 33111, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.Parse()
	addr := fmt.Sprintf("tcp://:%d", port)
	testCodecServe(addr, multicore, false, nil)
}
