package cable

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"

	pb "github.com/anycable/xk6-cable/ac_protos"
)

type Codec struct {
	Receive func(*websocket.Conn, interface{}) error
	Send    func(*websocket.Conn, interface{}) error
}

var JSONCodec = &Codec{
	Receive: func(c *websocket.Conn, v interface{}) error {
		return c.ReadJSON(v)
	},
	Send: func(c *websocket.Conn, v interface{}) error {
		return c.WriteJSON(v)
	},
}

var MsgPackCodec = &Codec{
	Receive: func(c *websocket.Conn, v interface{}) error {
		_, r, err := c.NextReader()
		if err != nil {
			return err
		}
		enc := msgpack.NewDecoder(r)
		enc.SetCustomStructTag("json")
		return enc.Decode(v)
	},
	Send: func(c *websocket.Conn, v interface{}) error {
		w, err := c.NextWriter(websocket.BinaryMessage)
		if err != nil {
			return err
		}
		enc := msgpack.NewEncoder(w)
		enc.SetCustomStructTag("json")
		err1 := enc.Encode(v)
		err2 := w.Close()
		if err1 != nil {
			return err1
		}
		return err2
	},
}

var ProtobufCodec = &Codec{
	Receive: func(c *websocket.Conn, v interface{}) error {
		mtype, r, err := c.NextReader()
		if err != nil {
			return err
		}

		if mtype != websocket.BinaryMessage {
			return fmt.Errorf("Unexpected message type: %v", mtype)
		}

		raw, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}

		buf := &pb.Message{}
		if err := proto.Unmarshal(raw, buf); err != nil {
			return err
		}

		msg := (v).(*cableMsg)

		msg.Type = buf.Type.String()
		msg.Identifier = buf.Identifier

		if buf.Message != nil {
			var message interface{}

			err = msgpack.Unmarshal(buf.Message, &message)
			msg.Message = message
		}

		return nil
	},
	Send: func(c *websocket.Conn, v interface{}) error {
		msg := (v).(*cableMsg)

		buf := &pb.Message{}

		buf.Command = pb.Command(pb.Command_value[msg.Command])
		buf.Identifier = msg.Identifier
		buf.Data = msg.Data

		b, err := proto.Marshal(buf)
		if err != nil {
			return err
		}

		w, err := c.NextWriter(websocket.BinaryMessage)
		if err != nil {
			return err
		}

		w.Write(b)
		err = w.Close()
		return err
	},
}
