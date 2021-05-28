package cable

import (
	"context"
	"errors"
	"fmt"
	"go.k6.io/k6/lib/metrics"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/stats"
)

type cableMsg struct {
	Type       string      `json:"type,omitempty"`
	Command    string      `json:"command,omitempty"`
	Identifier string      `json:"identifier,omitempty"`
	Data       string      `json:"data,omitempty"`
	Message    interface{} `json:"message,omitempty"`
}

type Client struct {
	ctx      context.Context
	codec    *Codec
	conn     *websocket.Conn
	channels map[string]*Channel

	readCh  chan *cableMsg
	errorCh chan error
	closeCh chan int

	mu         sync.Mutex
	logger     *logrus.Entry
	recTimeout time.Duration

	sampleTags *stats.SampleTags
	samplesOutput chan<- stats.SampleContainer
}

// Subscribe creates and returns Channel
func (c *Client) Subscribe(channelName string) (*Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	identifier := fmt.Sprintf("{\"channel\":\"%s\"}", channelName)

	if c.channels[identifier] != nil {
		c.logger.Errorf("already subscribed to `%v` channel\n", channelName)
		return c.channels[identifier], nil
	}

	if err := c.send(&cableMsg{Command: "subscribe", Identifier: identifier}); err != nil {
		return nil, err
	}

	channel := &Channel{client: c, identifier: identifier, readCh: make(chan *cableMsg), confCh: make(chan bool)}
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
			return nil, errors.New("subscription rejected")
		case <-timer:
			c.logger.Errorf("subscription to `%v`: timeout exceeded\n", channelName)
			return nil, errors.New("subscription timeout exceeded")
		}
	}
}

func (c *Client) send(msg *cableMsg) error {
	err := c.codec.Send(c.conn, msg)
	stats.PushIfNotDone(c.ctx, c.samplesOutput, stats.Sample{
		Metric: metrics.WSMessagesSent,
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
					c.channels[msg.Identifier].readCh <- msg
				}
			}
		case err := <-c.errorCh:
			// TODO: pass errors to the js script?
			c.logger.Errorf("websocket error: %v", err)
			continue
		case <-c.closeCh:
		case <-c.ctx.Done():
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
					continue
				}
			}
			code := websocket.CloseGoingAway
			if e, ok := err.(*websocket.CloseError); ok {
				code = e.Code
			}
			select {
			case c.closeCh <- code:
			}
			continue
		}

		select {
		case c.readCh <- obj:
			stats.PushIfNotDone(c.ctx, c.samplesOutput, stats.Sample{
				Metric: metrics.WSMessagesReceived,
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
