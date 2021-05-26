package cable

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
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

	mu sync.Mutex

	sampleTags *stats.SampleTags
}

// Subscribe creates and returns Channel
func (c *Client) Subscribe(channelName string) (*Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	identifier := fmt.Sprintf("{\"channel\":\"%s\"}", channelName)

	if c.channels[identifier] != nil {
		fmt.Printf("already subscribed to `%v` channel\n", channelName)
		return c.channels[identifier], nil
	}

	if err := c.send(&cableMsg{Command: "subscribe", Identifier: identifier}); err != nil {
		return nil, err
	}

	channel := &Channel{client: c, identifier: identifier, readCh: make(chan *cableMsg)}
	c.channels[identifier] = channel

	fmt.Printf("subscribed to `%v`\n", channelName)

	return channel, nil
}

func (c *Client) send(msg *cableMsg) error {
	return c.codec.Send(c.conn, msg)
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
				c.channels[msg.Identifier].readCh <- msg
			}
		case err := <-c.errorCh:
			// TODO: pass errors to the js script?
			fmt.Printf("Websocket error: %v", err)
			continue
		case <-c.closeCh:
		case <-c.ctx.Done():
			_ = c.conn.Close()
			// TODO: add debugging logs
			fmt.Println("quit")
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
			// TODO: increase message received k6 counter
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
		fmt.Printf("expected welcome msg, got %v", obj)
		panic(err)
	}
}

func (c *Client) receiveIgnoringPing() (*cableMsg, error) {
	for {
		var msg cableMsg
		if err := c.codec.Receive(c.conn, &msg); err != nil {
			return nil, err
		}

		if msg.Type == "ping" || msg.Type == "confirm_subscription" {
			continue
		}

		if msg.Type == "reject_subscription" {
			return nil, errors.New("subscription rejected")
		}

		return &msg, nil
	}
}
