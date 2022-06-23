package cable

import (
	"encoding/json"
	"go.k6.io/k6/js/modules"
	"sync"
	"time"

	"go.k6.io/k6/stats"

	"github.com/dop251/goja"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type cableMsg struct {
	Type       string      `json:"type,omitempty"`
	Command    string      `json:"command,omitempty"`
	Identifier string      `json:"identifier,omitempty"`
	Data       string      `json:"data,omitempty"`
	Message    interface{} `json:"message,omitempty"`
}

type Client struct {
	vu       modules.VU
	codec    *Codec
	conn     *websocket.Conn
	channels map[string]*Channel

	readCh  chan *cableMsg
	errorCh chan error
	closeCh chan int

	mu         sync.Mutex
	logger     *logrus.Entry
	recTimeout time.Duration

	sampleTags    *stats.SampleTags
	samplesOutput chan<- stats.SampleContainer
}

// Subscribe creates and returns Channel
func (c *Client) Subscribe(channelName string, paramsIn goja.Value) (*Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	params, err := c.parseParams(paramsIn)

	if err != nil {
		return nil, err
	}

	params["channel"] = channelName

	identifierJSON, err := json.Marshal(params)

	if err != nil {
		return nil, err
	}

	identifier := string(identifierJSON)

	if c.channels[identifier] != nil {
		c.logger.Warnf("already subscribed to `%v` channel\n", channelName)
		return c.channels[identifier], nil
	}

	if err := c.send(&cableMsg{Command: "subscribe", Identifier: identifier}); err != nil {
		return nil, err
	}

	channel := &Channel{
		client:     c,
		identifier: identifier,
		logger:     c.logger,
		readCh:     make(chan *cableMsg, 2048),
		confCh:     make(chan bool, 1),
	}
	c.channels[identifier] = channel

	timer := time.After(c.recTimeout)
	for {
		select {
		case confirmed := <-channel.confCh:
			if confirmed {
				c.logger.Debugf("subscribed to `%v`\n", channelName)
				return channel, nil
			}
			c.logger.Errorf("subscription to `%v`: rejected\n", channelName)
			return nil, nil
		case <-timer:
			c.logger.Errorf("subscription to `%v`: timeout exceeded. Consider increasing receiveTimeoutMs configuration option\n", channelName)
			return nil, nil
		}
	}
}

func (c *Client) Disconnect() {
	_ = c.conn.Close()
}

func (c *Client) send(msg *cableMsg) error {
	state := c.vu.State()
	if state == nil {
		return errCableInInitContext
	}

	err := c.codec.Send(c.conn, msg)
	stats.PushIfNotDone(c.vu.Context(), c.samplesOutput, stats.Sample{
		Metric: state.BuiltinMetrics.WSMessagesSent,
		Time:   time.Now(),
		Tags:   c.sampleTags,
		Value:  1,
	})

	return err
}

// start waits for the welcome message and then starts receive and handle loops.
func (c *Client) start() {
	c.receiveWelcomeMsg()
	go c.handleLoop()
	go c.receiveLoop()
}

func (c *Client) handleLoop() {
	for {
		select {
		case msg := <-c.readCh:
			if c.channels[msg.Identifier] != nil {
				switch msg.Type {
				case "confirm_subscription":
					c.channels[msg.Identifier].confCh <- true
				case "reject_subscription":
					c.channels[msg.Identifier].confCh <- false
				default:
					c.channels[msg.Identifier].handleIncoming(msg)
				}
			}
		case err := <-c.errorCh:
			c.logger.Errorf("websocket error: %v", err)
			continue
		case <-c.closeCh:
		case <-c.vu.Context().Done():
			_ = c.conn.Close()
			c.logger.Debugln("connection closed")
			return
		}
	}
}

func (c *Client) receiveLoop() {
	for {
		obj, err := c.receiveIgnoringPing()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				select {
				case c.errorCh <- err:
					return
				}
			}
			code := websocket.CloseGoingAway
			if e, ok := err.(*websocket.CloseError); ok {
				code = e.Code
			}
			select {
			case c.closeCh <- code:
			}
			return
		}

		if obj.Type == "disconnect" {
			c.logger.Debugln("connection closed by server")
			c.Disconnect()
			return
		}

		select {
		case c.readCh <- obj:
			state := c.vu.State()
			if state == nil {
				continue
			}
			stats.PushIfNotDone(c.vu.Context(), c.samplesOutput, stats.Sample{
				Metric: state.BuiltinMetrics.WSMessagesReceived,
				Time:   time.Now(),
				Tags:   c.sampleTags,
				Value:  1,
			})
			continue
		}
	}
}

func (c *Client) receiveWelcomeMsg() {
	obj, err := c.receiveIgnoringPing()
	if err != nil {
		panic(err)
	}

	if obj.Type != "welcome" {
		c.logger.Errorf("expected welcome msg, got %v", obj)

		panic(err)
	}
}

func (c *Client) receiveIgnoringPing() (*cableMsg, error) {
	for {
		var msg cableMsg
		if err := c.codec.Receive(c.conn, &msg); err != nil {
			return nil, err
		}
		c.logger.Debugf("message received: `%#v`\n", msg)

		if msg.Type == "ping" {
			continue
		}

		return &msg, nil
	}

}

func (c *Client) parseParams(in goja.Value) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	if in == nil || goja.IsUndefined(in) || goja.IsNull(in) {
		return params, nil
	}

	rt := c.vu.Runtime()
	data := in.ToObject(rt).Export().(map[string]interface{})

	return data, nil
}
