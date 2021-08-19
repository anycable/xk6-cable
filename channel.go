package cable

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/dop251/goja"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/common"
)

type Channel struct {
	client     *Client
	identifier string

	logger *logrus.Entry
	confCh chan bool
	readCh chan *cableMsg
}

// Perform sends passed action with additional data to the channel
func (ch *Channel) Perform(action string, attr goja.Value) error {
	rt := common.GetRuntime(ch.client.ctx)
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

// Receive checks channels messages query for message, sugar for ReceiveN(1, attrs)
func (ch *Channel) Receive(attr goja.Value) interface{} {
	results := ch.ReceiveN(1, attr)
	if len(results) == 0 {
		return nil
	}

	return results[0]
}

// ReceiveN checks channels messages query for provided number of messages satisfying provided condition.
func (ch *Channel) ReceiveN(n int, cond goja.Value) []interface{} {
	var results []interface{}
	timeout := ch.client.recTimeout
	timer := time.NewTimer(ch.client.recTimeout)
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
			ch.logger.Error("receive timeout exceeded; consider increasing receiveTimeoutMs configuration option")
			return results
		}
	}
}

type Matcher interface {
	Match(msg interface{}) bool
}

type FuncMatcher struct {
	rt *goja.Runtime
	f  goja.Callable
}

func (m *FuncMatcher) Match(msg interface{}) bool {
	result, err := m.f(goja.Undefined(), m.rt.ToValue(msg))

	if err != nil {
		panic(fmt.Sprintf("Can't call provided function: %v\n", err))
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

func (PassthruMatcher) Match(msg interface{}) bool {
	return true
}

// buildMatcher returns the corresponding matcher depending on condition type:
// - when condition is nil, match is always successful
// - when condition is a func, result of func(msg) is used as a result of match
// - when condition is a string, match is successful when message matches provided string
// - when condition is an object, match is successful when message includes all object attributes
func (ch *Channel) buildMatcher(cond goja.Value) (Matcher, error) {
	if cond == nil || goja.IsUndefined(cond) || goja.IsNull(cond) {
		return &PassthruMatcher{}, nil
	}

	if _, ok := cond.(*goja.Symbol); ok {
		return &StringMatcher{cond.String()}, nil
	}

	userFunc, isFunc := goja.AssertFunction(cond)

	if isFunc {
		return &FuncMatcher{common.GetRuntime(ch.client.ctx), userFunc}, nil
	}

	// we need to pass object through json unmarshalling to use same types for numbers
	jsonAttr, err := cond.ToObject(common.GetRuntime(ch.client.ctx)).MarshalJSON()
	if err != nil {
		return nil, err
	}

	var matcher map[string]interface{}
	_ = json.Unmarshal(jsonAttr, &matcher)

	return &AttrMatcher{matcher}, nil
}
