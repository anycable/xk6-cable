package cable

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/grafana/sobek"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/modules"
)

type Channel struct {
	client     *Client
	identifier string

	logger *logrus.Entry

	confCh chan bool
	ackMu  sync.Mutex
	readCh chan *cableMsg

	asyncHandlers []sobek.Callable

	ignoreReads bool

	createdAt time.Time
	ackedAt   time.Time
}

func NewChannel(c *Client, identifier string) *Channel {
	return &Channel{
		client:     c,
		identifier: identifier,
		logger:     c.logger,
		readCh:     make(chan *cableMsg, 2048),
		confCh:     make(chan bool, 1),
		createdAt:  time.Now(),
	}
}

// Perform sends passed action with additional data to the channel
func (ch *Channel) Perform(action string, attr sobek.Value) error {
	rt := ch.client.vu.Runtime()
	obj := attr.ToObject(rt).Export().(map[string]interface{})
	obj["action"] = action
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	return ch.client.send(&cableMsg{
		Command:    "message",
		Identifier: ch.identifier,
		Data:       string(data),
	})
}

// IgnoreReads allows skipping collecting incoming messages (in case you only care about the subscription)
func (ch *Channel) IgnoreReads() {
	ch.ignoreReads = true
}

// Receive checks channels messages query for message, sugar for ReceiveN(1, attrs)
func (ch *Channel) Receive(attr sobek.Value) interface{} {
	results := ch.ReceiveN(1, attr)
	if len(results) == 0 {
		return nil
	}

	return results[0]
}

// ReceiveN checks channels messages query for provided number of messages satisfying provided condition.
func (ch *Channel) ReceiveN(n int, cond sobek.Value) []interface{} {
	var results []interface{}
	timeout := ch.client.recTimeout
	timer := time.NewTimer(timeout)
	matcher, err := ch.buildMatcher(cond)
	if err != nil {
		panic(err)
	}

	i := 0
	for {
		select {
		case msg := <-ch.readCh:
			timer.Reset(timeout)
			if !matcher.Match(msg.Message) {
				continue
			}
			results = append(results, msg.Message)
			i++
			if i >= n {
				return results
			}
		case <-timer.C:
			ch.logger.Warn("receive timeout exceeded; consider increasing receiveTimeoutMs configuration option")
			return results
		}
	}
}

// ReceiveAll fethes all messages for a given number of seconds.
func (ch *Channel) ReceiveAll(sec int, cond sobek.Value) []interface{} {
	var results []interface{}
	timeout := time.Duration(sec) * time.Second
	timer := time.NewTimer(timeout)
	matcher, err := ch.buildMatcher(cond)
	if err != nil {
		panic(err)
	}

	for {
		select {
		case msg := <-ch.readCh:
			if !matcher.Match(msg.Message) {
				continue
			}
			results = append(results, msg.Message)
		case <-timer.C:
			return results
		}
	}
}

// Register callback to receive messages asynchronously
func (ch *Channel) OnMessage(fn sobek.Value) {
	f, isFunc := sobek.AssertFunction(fn)

	if !isFunc {
		panic("argument must be a function")
	}

	ch.asyncHandlers = append(ch.asyncHandlers, f)
}

func (ch *Channel) AckDuration() int64 {
	ch.ackMu.Lock()
	defer ch.ackMu.Unlock()

	return ch.ackedAt.Sub(ch.createdAt).Milliseconds()
}

func (ch *Channel) handleAck(val bool, when time.Time) {
	ch.ackMu.Lock()
	defer ch.ackMu.Unlock()

	ch.ackedAt = when
	ch.confCh <- val
}

func (ch *Channel) handleIncoming(msg *cableMsg) {
	ch.handleAsync(msg)

	if ch.ignoreReads {
		return
	}

	ch.readCh <- msg
}

func (ch *Channel) handleAsync(msg *cableMsg) {
	if msg == nil {
		return
	}

	ch.client.mu.Lock()
	defer ch.client.mu.Unlock()

	if ch.client.disconnected {
		return
	}

	for _, h := range ch.asyncHandlers {
		_, err := h(sobek.Undefined(), ch.client.vu.Runtime().ToValue(msg.Message))
		if err != nil {
			if !strings.Contains(err.Error(), "context canceled") {
				ch.logger.Errorf("can't call provided function: %s", err)
			}
		}
	}
}

type Matcher interface {
	Match(msg interface{}) bool
}

type FuncMatcher struct {
	vu modules.VU
	f  sobek.Callable
}

func (m *FuncMatcher) Match(msg interface{}) bool {
	result, err := m.f(sobek.Undefined(), m.vu.Runtime().ToValue(msg))
	if err != nil {
		m.vu.State().Logger.Errorf("can't call provided function: %v", err)
	}

	return result.ToBoolean()
}

type StringMatcher struct {
	expected string
}

func (m *StringMatcher) Match(msg interface{}) bool {
	msgStr, ok := msg.(string)

	if !ok {
		return false
	}

	return m.expected == msgStr
}

type AttrMatcher struct {
	expected map[string]interface{}
}

func (m *AttrMatcher) Match(msg interface{}) bool {
	msgObj, ok := msg.(map[string]interface{})
	if !ok {
		return false
	}

	for k, v := range m.expected {
		if !reflect.DeepEqual(v, msgObj[k]) {
			return false
		}
	}

	return true
}

type PassthruMatcher struct{}

func (PassthruMatcher) Match(_ interface{}) bool {
	return true
}

// buildMatcher returns the corresponding matcher depending on condition type:
// - when condition is nil, match is always successful
// - when condition is a func, result of func(msg) is used as a result of match
// - when condition is a string, match is successful when message matches provided string
// - when condition is an object, match is successful when message includes all object attributes
func (ch *Channel) buildMatcher(cond sobek.Value) (Matcher, error) {
	if cond == nil || sobek.IsUndefined(cond) || sobek.IsNull(cond) {
		return &PassthruMatcher{}, nil
	}

	if _, ok := cond.(*sobek.Symbol); ok {
		return &StringMatcher{cond.String()}, nil
	}

	userFunc, isFunc := sobek.AssertFunction(cond)

	if isFunc {
		return &FuncMatcher{ch.client.vu, userFunc}, nil
	}

	// we need to pass object through json unmarshalling to use same types for numbers
	jsonAttr, err := cond.ToObject(ch.client.vu.Runtime()).MarshalJSON()
	if err != nil {
		return nil, err
	}

	var matcher map[string]interface{}
	_ = json.Unmarshal(jsonAttr, &matcher)

	return &AttrMatcher{matcher}, nil
}
