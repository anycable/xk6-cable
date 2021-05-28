package cable

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/dop251/goja"
	"github.com/gorilla/websocket"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/lib/metrics"
	"go.k6.io/k6/stats"
)

func init() {
	modules.Register("k6/x/cable", new(Cable))
}

// errCableInInitContext is returned when cable used in the init context
var errCableInInitContext = common.NewInitContextError("using cable in the init context is not supported")

type Cable struct{}

// Connect connects to the websocket, creates and starts client, and returns it to the js.
func (r *Cable) Connect(ctx context.Context, url string, opts goja.Value) *Client {
	state := lib.GetState(ctx)
	if state == nil {
		panic(errCableInInitContext)
	}

	cOpts, err := parseOptions(ctx, opts)
	if err != nil {
		panic(err)
	}

	wsd := createDialer(state, cOpts.handshakeTimeout())
	connectionStart := time.Now()
	conn, httpResponse, connErr := wsd.DialContext(ctx, url, cOpts.header())
	connectionEnd := time.Now()

	tags := cOpts.appendTags(state.CloneTags())
	if state.Options.SystemTags.Has(stats.TagIP) && conn != nil && conn.RemoteAddr() != nil {
		if ip, _, err := net.SplitHostPort(conn.RemoteAddr().String()); err == nil {
			tags["ip"] = ip
		}
	}
	if httpResponse != nil {
		if state.Options.SystemTags.Has(stats.TagStatus) {
			tags["status"] = strconv.Itoa(httpResponse.StatusCode)
		}

		if state.Options.SystemTags.Has(stats.TagSubproto) {
			tags["subproto"] = httpResponse.Header.Get("Sec-WebSocket-Protocol")
		}
	}
	if state.Options.SystemTags.Has(stats.TagURL) {
		tags["url"] = url
	}

	sampleTags := stats.IntoSampleTags(&tags)

	stats.PushIfNotDone(ctx, state.Samples, stats.ConnectedSamples{
		Samples: []stats.Sample{
			{Metric: metrics.WSSessions, Time: connectionStart, Tags: sampleTags, Value: 1},
			{Metric: metrics.WSConnecting, Time: connectionStart, Tags: sampleTags, Value: stats.D(connectionEnd.Sub(connectionStart))},
		},
		Tags: sampleTags,
		Time: connectionStart,
	})

	if connErr != nil {
		panic(connErr)
	}

	client := Client{
		ctx:           ctx,
		conn:          conn,
		codec:         cOpts.codec(),
		logger:        state.Logger.WithField("source", "cable"),
		channels:      make(map[string]*Channel),
		readCh:        make(chan *cableMsg),
		errorCh:       make(chan error),
		closeCh:       make(chan int),
		recTimeout:    cOpts.recTimeout(),
		sampleTags:    sampleTags,
		samplesOutput: state.Samples,
	}

	client.start()

	return &client
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
