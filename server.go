package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/leesper/holmes"
	"github.com/nsqio/go-nsq"
	"github.com/sapariduo/fmxxx/handlers"
	ts "github.com/sapariduo/teleserver"
)

var (
	port   = flag.Int("port", 11000, "Port")
	debug  = flag.Bool("debug", false, "use log debug mode")
	mqhost = flag.String("mqhost", "", "message queue host")
	mqport = flag.String("mqport", "", "mesage queue port")
	// Pub   *nsq.Producer
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
	codec := ts.CustomCodecOption(handlers.FMXXXCodec{})

	workers := ts.WorkerSizeOption(100)

	return &FMXXXServer{
		ts.NewServer(onConnect, onClose, onError, onMessage, codec, workers),
	}
}

func init() {
	flag.Parse()
	if *mqhost == "" || *mqport == "" {
		flag.Usage()
		os.Exit(0)
	}
	conf := nsq.NewConfig()
	nsqAddr := *mqhost + ":" + *mqport
	// conf.ReadTimeout = (10 * time.Second)
	conf.MaxAttempts = 5
	prod, err := nsq.NewProducer(nsqAddr, conf)
	if err != nil {
		holmes.Errorln(err)
		holmes.Fatalln("Could not create producer, Message Bus not found")
		os.Exit(0)
	}
	err = prod.Ping()
	if err != nil {
		os.Exit(0)
	}
	handlers.Pub = prod
}

func main() {
	// flag.Parse()
	if *debug {
		defer holmes.Start(holmes.DebugLevel).Stop()
	} else {
		defer holmes.Start(holmes.ErrorLevel, holmes.InfoLevel).Stop()
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	ts.MonitorOn(*port + 10000)
	ts.Register(handlers.Message{}.MessageNumber(), handlers.DeserializeMessage, handlers.ProcessMessage)
	netport := ":" + strconv.Itoa(*port)
	l, err := net.Listen("tcp", netport)
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
