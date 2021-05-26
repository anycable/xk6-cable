package cable

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/dop251/goja"
	"go.k6.io/k6/js/common"
)

type Channel struct {
	client     *Client
	identifier string

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
func (ch *Channel) Receive(attr goja.Value) (interface{}, error) {
	results, err := ch.ReceiveN(1, attr)
	if len(results) == 0 {
		return nil, err
	}

	return results[0], err
}

// ReceiveN checks channels messages query for provided number of messages satisfying provided condition.
func (ch *Channel) ReceiveN(n int, cond goja.Value) ([]interface{}, error) {
	var results []interface{}
	timer := time.After(300 * time.Millisecond) // TODO: configurable

	i := 0
	for {
		select {
		case msg := <-ch.readCh:
			if !ch.matches(msg.Message, cond) {
				continue
			}
			results = append(results, msg.Message)
			i++
			if i >= n {
				return results, nil
			}
		case <-timer:
			return nil, errors.New("receiving timeout exceeded")
		}
	}
}

// matches used to check passed message against provided condition:
// - when condition is nil, match is always successful
// - when condition is func, result of func(msg) is used as a result of match
// - when condition is object, match is successful when message includes all object attributes
func (ch *Channel) matches(msg interface{}, cond goja.Value) bool {
	if cond == nil || goja.IsUndefined(cond) || goja.IsNull(cond) {
		return true
	}

	// TODO: seems like it will never be used?
	if _, ok := cond.(*goja.Symbol); ok {
		if _, ok := msg.(string); !ok {
			return false
		}

		return cond.String() == msg.(string)
	}

	rt := common.GetRuntime(ch.client.ctx)
	userFunc, isFunc := goja.AssertFunction(cond)
	if isFunc {
		result, err := userFunc(goja.Undefined(), rt.ToValue(msg))
		if err != nil {
			panic(fmt.Sprintf("Can't call provided function: %v\n", err))
		}

		return result.ToBoolean()
	}

	msgObj, ok := msg.(map[string]interface{})
	if !ok {
		return false
	}
	// we need to pass object through json unmarshalling to use same types for numbers
	jsonAttr, err := cond.ToObject(rt).MarshalJSON()
	if err != nil {
		panic(err)
	}
	var matcher map[string]interface{}
	_ = json.Unmarshal(jsonAttr, &matcher)

	for k, v := range matcher {
		if !reflect.DeepEqual(v, msgObj[k]) {
			return false
		}
	}

	return true
}
