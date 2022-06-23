package cable

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dop251/goja"
)

type connectOptions struct {
	Cookies string            `json:"cookies"`
	Headers map[string]string `json:"headers"`
	Tags    map[string]string `json:"tags"`
	Codec   string            `json:"codec"`

	HandshakeTimeoutS int    `json:"handshakeTimeoutS"`
	ReceiveTimeoutMs  int    `json:"receiveTimeoutMs"`
	LogLevel          string `json:"logLevel"`
}

const (
	defaultHandshakeTimeout = 60
	defaultReceiveTimeout   = 1000
)

func parseOptions(rt *goja.Runtime, inOpts goja.Value) (*connectOptions, error) {
	var outOpts connectOptions

	if inOpts == nil || goja.IsUndefined(inOpts) || goja.IsNull(inOpts) {
		return &outOpts, nil
	}

	data, err := json.Marshal(inOpts.ToObject(rt).Export())
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&outOpts); err != nil {
		if uerr := json.Unmarshal(data, &outOpts); uerr != nil {
			return nil, uerr
		}
		return nil, err
	}
	return &outOpts, nil
}

func (co *connectOptions) codec() *Codec {
	if co.Codec == "msgpack" {
		return MsgPackCodec
	} else if co.Codec == "protobuf" {
		return ProtobufCodec
	}

	return JSONCodec
}

func (co *connectOptions) handshakeTimeout() time.Duration {
	if co.HandshakeTimeoutS == 0 {
		return defaultHandshakeTimeout * time.Second
	}

	return time.Duration(co.HandshakeTimeoutS) * time.Second
}

func (co *connectOptions) receiveTimeout() time.Duration {
	if co.ReceiveTimeoutMs == 0 {
		return defaultReceiveTimeout * time.Millisecond
	}

	return time.Duration(co.ReceiveTimeoutMs) * time.Millisecond
}

func (co *connectOptions) appendTags(tags map[string]string) map[string]string {
	if len(co.Tags) > 0 {
		for k, v := range co.Tags {
			tags[k] = v
		}
	}
	return tags
}

func (co *connectOptions) header() http.Header {
	header := http.Header{}

	if len(co.Headers) > 0 {
		for k, v := range co.Headers {
			header.Set(k, v)
		}
	}

	if co.Cookies != "" {
		header.Set("COOKIE", co.Cookies)
	}

	return header
}
