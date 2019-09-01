package main


import (
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/leesper/holmes"
	"github.com/sapariduo/fmxxx/handlers"
	ts "github.com/sapariduo/teleserver"
)

// EchoServer represents the echo server.
type FMXXXServer struct {
	*ts.Server
}

// NewEchoServer returns an EchoServer.
func NewFMServer() *FMXXXServer {
	onConnect := ts.OnConnectOption(func(conn ts.WriteCloser) bool {
		_, _, sec := time.Now().Clock()
		conn.(*ts.ServerConn).SetContextValue("codec", sec)
		holmes.Infoln("on connect")
		return true
	})

	onClose := ts.OnCloseOption(func(conn ts.WriteCloser) {
		holmes.Infoln("closing client")
	})

	onError := ts.OnErrorOption(func(conn ts.WriteCloser) {
		holmes.Infoln("on error")
		b := []byte{0} // 0x00 if we decline the message
		resp := handlers.Response{Content: b}
		conn.Write(resp)
	})

	onMessage := ts.OnMessageOption(func(msg ts.Message, conn ts.WriteCloser) {
		holmes.Infoln("receving message")
	})
	codec := ts.CustomCodecOption(handlers.FreeTypeCodec{})

	workers := ts.WorkerSizeOption(25)

	return &FMXXXServer{
		ts.NewServer(onConnect, onClose, onError, onMessage, codec, workers),
	}
}

func main() {
	defer holmes.Start(holmes.ErrorLevel, holmes.InfoLevel).Stop()

	runtime.GOMAXPROCS(runtime.NumCPU())
	ts.MonitorOn(11000)
	ts.Register(handlers.Message{}.MessageNumber(), handlers.DeserializeMessage, handlers.ProcessMessage)

	l, err := net.Listen("tcp", ":12345")
	if err != nil {
		holmes.Fatalf("listen error %v", err)
	}
	fmxxxserver := NewFMServer()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		fmxxxserver.Stop()
	}()

	fmxxxserver.Start(l)
}
