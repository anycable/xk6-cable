package cable

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/metrics"

	"github.com/gorilla/websocket"
	"github.com/grafana/sobek"
	"github.com/sirupsen/logrus"
)

type cableMsg struct {
	Type       string      `json:"type,omitempty"`
	Command    string      `json:"command,omitempty"`
	Identifier string      `json:"identifier,omitempty"`
	Data       string      `json:"data,omitempty"`
	Message    interface{} `json:"message,omitempty"`

	receivedAt time.Time
}

type Client struct {
	vu       modules.VU
	codec    *Codec
	conn     *websocket.Conn
	channels map[string]*Channel

	readCh  chan *cableMsg
	errorCh chan error
	closeCh chan int

	disconnected bool

	mu         sync.Mutex
	logger     *logrus.Entry
	recTimeout time.Duration

	sampleTags    *metrics.TagSet
	samplesOutput chan<- metrics.SampleContainer
}

// Subscribe creates and returns Channel
func (c *Client) Subscribe(channelName string, paramsIn sobek.Value) (*Channel, error) {
	promise, err := c.SubscribeAsync(channelName, paramsIn)
	if err != nil {
		return nil, err
	}

	channel, err := promise.Await(int(c.recTimeout.Milliseconds()))

	if err != nil {
		return nil, err
	} else {
		return channel, nil
	}
}

type SubscribePromise struct {
	client  *Client
	channel *Channel
}

func (sp *SubscribePromise) Await(ms int) (*Channel, error) {
	if ms == 0 {
		ms = int(sp.client.recTimeout.Milliseconds())
	}

	timer := time.After(time.Duration(ms) * time.Millisecond)

	select {
	case confirmed := <-sp.channel.confCh:
		if confirmed {
			sp.client.logger.Debugf("subscribed to `%v`\n", sp.channel.identifier)
			return sp.channel, nil
		}
		return nil, fmt.Errorf("subscription to `%v`: rejected", sp.channel.identifier)
	case <-timer:
		return nil, fmt.Errorf("subscription to `%v`: timeout exceeded. Consider increasing receiveTimeoutMs configuration option (current: %d)", sp.channel.identifier, ms)
	}
}

// Subscribe creates and returns Channel
func (c *Client) SubscribeAsync(channelName string, paramsIn sobek.Value) (*SubscribePromise, error) {
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
		return &SubscribePromise{client: c, channel: c.channels[identifier]}, nil
	}

	channel := NewChannel(c, identifier)

	if err := c.send(&cableMsg{Command: "subscribe", Identifier: identifier}); err != nil {
		return nil, err
	}

	c.channels[identifier] = channel

	return &SubscribePromise{client: c, channel: channel}, nil
}

func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.disconnected {
		return
	}

	c.disconnected = true
	_ = c.conn.Close()
}

// Repeat function in a loop until it returns false
func (c *Client) Loop(fn sobek.Value) {
	f, isFunc := sobek.AssertFunction(fn)

	if !isFunc {
		panic("argument must be a function")
	}

	for {
		select {
		case <-c.vu.Context().Done():
			c.Disconnect()
			return
		default:
			c.mu.Lock()
			ret, err := f(sobek.Undefined())
			c.mu.Unlock()

			if err != nil {
				if !strings.Contains(err.Error(), "context canceled") {
					c.logger.Errorf("loop execution failed: %v", err)
				}
				return
			}

			c.mu.Lock()
			result := ret.ToBoolean()
			c.mu.Unlock()

			if result {
				return
			}
		}
	}
}

func (c *Client) send(msg *cableMsg) error {
	state := c.vu.State()
	if state == nil {
		return errCableInInitContext
	}

	err := c.codec.Send(c.conn, msg)
	metrics.PushIfNotDone(c.vu.Context(), c.samplesOutput, metrics.Sample{
		TimeSeries: metrics.TimeSeries{
			Metric: state.BuiltinMetrics.WSMessagesSent,
			Tags:   c.sampleTags,
		},
		Time:  time.Now(),
		Value: 1,
	})

	return err
}

// start waits for the welcome message and then starts receive and handle loops.
func (c *Client) start() error {
	err := c.receiveWelcomeMsg()
	if err != nil {
		return err
	}

	go c.handleLoop()
	go c.receiveLoop()

	return nil
}

func (c *Client) handleLoop() {
	for {
		select {
		case msg := <-c.readCh:
			if c.channels[msg.Identifier] != nil {
				switch msg.Type {
				case "confirm_subscription":
					c.channels[msg.Identifier].handleAck(true, msg.receivedAt)
				case "reject_subscription":
					c.channels[msg.Identifier].handleAck(false, msg.receivedAt)
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
			metrics.PushIfNotDone(c.vu.Context(), c.samplesOutput, metrics.Sample{
				TimeSeries: metrics.TimeSeries{
					Metric: state.BuiltinMetrics.WSMessagesReceived,
					Tags:   c.sampleTags,
				},
				Time:  time.Now(),
				Value: 1,
			})
			continue
		}
	}
}

func (c *Client) receiveWelcomeMsg() error {
	obj, err := c.receiveIgnoringPing()
	if err != nil {
		return err
	}

	if obj.Type != "welcome" {
		return fmt.Errorf("expected welcome msg, got %v", obj)
	}

	return nil
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

		msg.receivedAt = time.Now()

		timestamp := int64(time.Now().UnixNano()) / 1_000_000

		if data, ok := msg.Message.(map[string]interface{}); ok {
			data["__timestamp__"] = timestamp
			msg.Message = data
		}

		return &msg, nil
	}
}

func (c *Client) parseParams(in sobek.Value) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	if in == nil || sobek.IsUndefined(in) || sobek.IsNull(in) {
		return params, nil
	}

	rt := c.vu.Runtime()
	data := in.ToObject(rt).Export().(map[string]interface{})

	return data, nil
}
