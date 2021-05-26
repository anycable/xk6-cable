package cable

import (
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
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
