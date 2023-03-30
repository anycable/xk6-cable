package cable

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.k6.io/k6/js/common"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/metrics"

	"github.com/dop251/goja"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// errCableInInitContext is returned when cable used in the init context
var errCableInInitContext = common.NewInitContextError("using cable in the init context is not supported")

// Connect connects to the websocket, creates and starts client, and returns it to the js.
func (c *Cable) Connect(cableUrl string, opts goja.Value) (*Client, error) {
	state := c.vu.State()
	if state == nil {
		return nil, errCableInInitContext
	}

	cOpts, err := parseOptions(c.vu.Runtime(), opts)
	if err != nil {
		return nil, err
	}

	wsd := createDialer(state, cOpts.handshakeTimeout())
	connectionStart := time.Now()

	headers := cOpts.header()

	if headers.Get("ORIGIN") == "" {
		uri, perr := url.Parse(cableUrl)

		if perr == nil {
			var scheme string

			if uri.Scheme == "wss" {
				scheme = "https"
			} else {
				scheme = "http"
			}

			origin := fmt.Sprintf("%s://%s", scheme, uri.Host)

			headers.Set("ORIGIN", origin)
		}
	}

	if cOpts.codec() == JSONCodec {
		headers.Set("Sec-WebSocket-Protocol", "actioncable-v1-json")
	} else if cOpts.codec() == MsgPackCodec {
		headers.Set("Sec-WebSocket-Protocol", "actioncable-v1-msgpack")
	} else if cOpts.codec() == ProtobufCodec {
		headers.Set("Sec-WebSocket-Protocol", "actioncable-v1-protobuf")
	}

	level, err := logrus.ParseLevel(cOpts.LogLevel)

	if err == nil {
		state.Logger.SetLevel(level)
	}

	logger := state.Logger.WithField("source", "cable")

	conn, httpResponse, connErr := wsd.DialContext(c.vu.Context(), cableUrl, headers)
	connectionEnd := time.Now()

	tagsAndMeta := state.Tags.GetCurrentValues()

	if state.Options.SystemTags.Has(metrics.TagIP) && conn != nil && conn.RemoteAddr() != nil {
		if ip, _, err := net.SplitHostPort(conn.RemoteAddr().String()); err == nil {
			tagsAndMeta.SetSystemTagOrMeta(metrics.TagIP, ip)
		}
	}
	if httpResponse != nil {
		if state.Options.SystemTags.Has(metrics.TagStatus) {
			tagsAndMeta.SetSystemTagOrMeta(metrics.TagStatus, strconv.Itoa(httpResponse.StatusCode))
		}

		if state.Options.SystemTags.Has(metrics.TagSubproto) {
			tagsAndMeta.SetSystemTagOrMeta(metrics.TagSubproto, httpResponse.Header.Get("Sec-WebSocket-Protocol"))
		}
	}
	if state.Options.SystemTags.Has(metrics.TagURL) {
		tagsAndMeta.SetSystemTagOrMetaIfEnabled(state.Options.SystemTags, metrics.TagURL, cableUrl)
	}

	metrics.PushIfNotDone(c.vu.Context(), state.Samples, metrics.ConnectedSamples{
		Samples: []metrics.Sample{
			{TimeSeries: metrics.TimeSeries{
				Metric: state.BuiltinMetrics.WSSessions,
				Tags:   tagsAndMeta.Tags,
			},
				Time:     connectionStart,
				Metadata: tagsAndMeta.Metadata,
				Value:    1},
			{TimeSeries: metrics.TimeSeries{
				Metric: state.BuiltinMetrics.WSConnecting,
				Tags:   tagsAndMeta.Tags,
			},
				Time:     connectionStart,
				Metadata: tagsAndMeta.Metadata,
				Value:    metrics.D(connectionEnd.Sub(connectionStart))},
		},
		Tags: tagsAndMeta.Tags,
		Time: connectionStart,
	})

	if connErr != nil {
		logger.Errorf("failed to connect: %v", connErr)
		return nil, nil
	}

	client := Client{
		vu:            c.vu,
		conn:          conn,
		codec:         cOpts.codec(),
		logger:        logger,
		channels:      make(map[string]*Channel),
		readCh:        make(chan *cableMsg, 1024),
		errorCh:       make(chan error, 1024),
		closeCh:       make(chan int, 1),
		recTimeout:    cOpts.receiveTimeout(),
		sampleTags:    tagsAndMeta.Tags,
		samplesOutput: state.Samples,
	}

	err = client.start()

	if err != nil {
		return nil, err
	}

	return &client, nil
}

func createDialer(state *lib.State, handshakeTimeout time.Duration) websocket.Dialer {
	// Overriding the NextProtos to avoid talking http2
	var tlsConfig *tls.Config
	if state.TLSConfig != nil {
		tlsConfig = state.TLSConfig.Clone()
		tlsConfig.NextProtos = []string{"http/1.1"}
	}

	wsd := websocket.Dialer{
		HandshakeTimeout: handshakeTimeout,
		// Pass a custom net.DialContext function to websocket.Dialer that will substitute
		// the underlying net.Conn with k6 tracked netext.Conn
		NetDialContext:  state.Dialer.DialContext,
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}
	return wsd
}
