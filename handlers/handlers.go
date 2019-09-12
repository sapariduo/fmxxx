package handlers

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"net"

	"github.com/leesper/holmes"
	"github.com/nsqio/go-nsq"
	ts "github.com/sapariduo/teleserver"
)

var (
	Pub *nsq.Producer
)

const (
	// MessageTypeBytes is the length of type header.
	MessageTypeBytes = 4
	// MessageLenBytes is the length of length header.
	MessageLenBytes = 4
	// MessageMaxBytes is the maximum bytes allowed for application data.
	MessageMaxBytes = 1 << 10 // 1024

	MsgType = 5
)

// var MessageLenBytes int

//Message is entity for incoming message
type Message struct {
	// Content string
	Content []byte
}

// Serialize serializes Message into bytes.
func (em Message) Serialize() ([]byte, error) {
	return em.Content, nil
}

// MessageNumber returns message type number.
func (em Message) MessageNumber() int32 {
	return 5
}

type Response struct {
	Content []byte
}

func (res Response) Serialize() ([]byte, error) {
	return res.Content, nil
}

func (res Response) MessageNumber() int32 {
	return 5
}

// DeserializeMessage deserializes bytes into Message.
func DeserializeMessage(data []byte) (message ts.Message, err error) {
	if data == nil {
		return nil, ts.ErrNilData
	}
	msg := data
	payload := Message{
		Content: msg,
	}
	return payload, nil
}

// ProcessMessage process the logic of echo message.
func ProcessMessage(ctx context.Context, conn ts.WriteCloser) {
	msg := ts.MessageFromContext(ctx).(Message)
	c := conn.(*ts.ServerConn)
	netid := c.RemoteAddr().String()
	c.SetContextValue("netid", netid)
	holmes.Debugf("receving message %x from %s\n ", msg.Content, netid)
	buff := new(bytes.Buffer)
	imei := c.ContextValue("imei")
	if imei == nil {
		body := msg.Content
		size := len(body)
		strimei := hex.EncodeToString(body[:size])
		c.SetContextValue("imei", strimei)
		holmes.Infof("Received message from %s with size %d", strimei, size)
		buff.Write([]byte{1})
		res := Response{Content: buff.Bytes()}
		conn.Write(res)
	} else {
		buf := msg.Content
		size := len(buf)
		stringbuf := hex.EncodeToString(buf[:size])
		holmes.Infof("Received message from %s with size %d", c.ContextValue("imei"), size)
		holmes.Debugf(" Received imei %s, size %d, message %s\n", imei, size, stringbuf)
		elements, err := parseData(buf, size, imei.(string))
		if err != nil {
			holmes.Errorf("Error while parsing data %x \n", err)
			buff.Write([]byte{0})
			res := Response{Content: buff.Bytes()}
			conn.Write(res)
			conn.Close()
		}

		for i := 0; i < len(elements); i++ {
			element := elements[i]
			// err := rc.Insert(&element)
			// if err != nil {
			// 	fmt.Println("Error inserting element to database", err)
			// }
			js, _ := json.Marshal(element)
			err = Pub.Publish("teltonika", js)
			if err != nil {
				holmes.Errorln("Error inserting element to Message Bus", err)
			}
			holmes.Debugf("%s", string(js))
		}
		resp := []byte{0, 0, 0, uint8(len(elements))}

		buff.Write(resp)
		res := Response{Content: buff.Bytes()}
		conn.Write(res)
	}

}

type FMXXXCodec struct{}

// Decode decodes the bytes data into Message
func (codec FMXXXCodec) Decode(raw net.Conn) (ts.Message, error) {
	byteChan := make(chan []byte)
	errorChan := make(chan error)

	go func(bc chan []byte, ec chan error) {
		typeData := make([]byte, MessageMaxBytes)
		// _, err := io.ReadFull(raw, typeData)
		ln, err := raw.Read(typeData)
		// ln, err := raw.
		if err != nil {
			ec <- err
			close(bc)
			close(ec)
			holmes.Debugln("go-routine read message type exited")
			return
		}
		bc <- typeData[:ln]
	}(byteChan, errorChan)

	var typeBytes []byte

	select {
	case err := <-errorChan:
		return nil, err

	case typeBytes = <-byteChan:
		if typeBytes == nil {
			holmes.Warnln("read type bytes nil")
			return nil, ts.ErrBadData
		}

		// deserialize message from bytes
		unmarshaler := ts.GetUnmarshalFunc(MsgType)
		if unmarshaler == nil {
			return nil, ts.ErrUndefined(MsgType)
		}
		return unmarshaler(typeBytes)
	}
}

// Encode encodes the message into bytes data.
func (codec FMXXXCodec) Encode(msg ts.Message) ([]byte, error) {
	data, err := msg.Serialize()
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)

	// binary.Write(buf, binary.LittleEndian, msg.MessageNumber())
	// binary.Write(buf, binary.LittleEndian, int16(len(data)))
	buf.Write(data)
	packet := buf.Bytes()
	return packet, nil
}
