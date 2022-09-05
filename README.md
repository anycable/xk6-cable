# xk6-cable

A k6 extension for testing Action Cable and AnyCable functionality. Built for [k6][] using the [xk6][] system.

Comparing to the official [WebSockets support][k6-websockets], `xk6-cable` provides the following features:

- Built-in Action Cable API support (no need to manually build or parse protocol messages).
- Synchronous API to initialize connections and subscriptions.
- (WIP) AnyCable-specific extensions (e.g., binary encodings)

> Read also ["Real-time stress: AnyCable, k6, WebSockets, and Yabeda"](https://evilmartians.com/chronicles/real-time-stress-anycable-k6-websockets-and-yabeda?utm_source=xk6-cable-github)

## Build

To build a `k6` binary with this extension, first ensure you have the prerequisites:

- [Go toolchain](https://go101.org/article/go-toolchain.html) v1.17+
- Git

1. Install `xk6` framework for extending `k6`:

```sh
go install go.k6.io/xk6/cmd/xk6@latest
```

2. Build the binary:

```shell
xk6 build --with github.com/anycable/xk6-cable@latest

# you can specify k6 version
xk6 build v0.38.3 --with github.com/anycable/xk6-cable@latest

# or if you want to build from the local source
xk6 build --with github.com/anycable/xk6-cable@latest=/path/to/source
```

## Example

Consider a simple example using the EchoChannel:

```js
// benchmark.js
import { check, sleep } from 'k6';
import cable from "k6/x/cable";

export default function () {
  // Initialize the connection
  const client = cable.connect("ws://localhost:8080/cable");
  // If connection were not sucessful, the return value is null
  // It's a good practice to add a check and configure a threshold (so, you can fail-fast if
  // configuration is incorrect)
  if (
    !check(client, {
      "successful connection": (obj) => obj,
    })
  ) {
    fail("connection failed");
  }

  // At this point, the client has been successfully connected
  // (e.g., welcome message has been received)

  // Send subscription request and wait for the confirmation.
  // Returns null if failed to subscribe (due to rejection or timeout).
  const channel = client.subscribe("EchoChannel");

  // Perform an action
  channel.perform("echo", { foo: 1 });

  // Retrieve a single message from the incoming inbox (FIFO).
  // Returns null if no messages have been received in the specified period of time (see below).
  const res = channel.receive();
  check(res, {
    "received res": (obj) => obj.foo === 1,
  });

  channel.perform("echo", { foobar: 3 });
  channel.perform("echo", { foobaz: 3 });

  // You can also retrieve multiple messages at a time.
  // Returns as many messages (but not more than expected) as have been received during
  // the specified period of time. If none, returns an empty array.
  const reses = channel.receiveN(2);
  check(reses, {
    "received 2 messages": (obj) => obj.length === 2,
  });

  sleep(1);

  // Terminate the WS connection
  client.disconnect()
}
```

Example run results:

```sh
$ ./k6 run benchmark.js


          /\      |‾‾| /‾‾/   /‾‾/
     /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
   /          \   |  |\  \ |  (‾)  |
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: benchmark.js
     output: -

  scenarios: (100.00%) 1 scenario, 1 max VUs, 10m30s max duration (incl. graceful stop):
           * default: 1 iterations for each of 1 VUs (maxDuration: 10m0s, gracefulStop: 30s)


running (00m00.0s), 0/1 VUs, 1 complete and 0 interrupted iterations
default ✓ [======================================] 1 VUs  00m00.0s/10m0s  1/1 iters, 1 per VU

     ✓ received res
     ✓ received res2
     ✓ received 3 messages
     ✓ received 2 messages
     ✓ all messages with baz attr

     checks...............: 100.00% ✓ 5 ✗ 0
     data_received........: 995 B   83 kB/s
     data_sent............: 1.2 kB  104 kB/s
     iteration_duration...: avg=11.06ms  min=11.06ms  med=11.06ms  max=11.06ms  p(90)=11.06ms  p(95)=11.06ms
     iterations...........: 1       83.850411/s
     ws_connecting........: avg=904.62µs min=904.62µs med=904.62µs max=904.62µs p(90)=904.62µs p(95)=904.62µs
     ws_msgs_received.....: 9       754.653698/s
     ws_msgs_sent.........: 9       754.653698/s
     ws_sessions..........: 1       83.850411/s
```

You can pass the following options to the `connect` method as the second argument:

```js
{
  headers: {}, // HTTP headers to use (e.g., { COOKIE: 'some=cookie;' })
  cookies: "", // HTTP cookies as string (overwrite the value passed in headers if present)
  tags: {}, // k6 tags
  handshakeTimeoutS: 60, // Max allowed time to initialize a connection
  receiveTimeoutMs: 1000, // Max time to wait for an incoming message
  logLevel: "info" // logging level (change to debug to see more information)
  codec: "json", // Codec (encoder) to use. Supported values are: json, msgpack, protobuf.
}
```

**NOTE:** `msgpack` and `protobuf` codecs are only supported by [AnyCable PRO](https://anycable.io#pro).

More examples could be found in the [examples/](./examples) folder.

## Contributing

Bug reports and pull requests are welcome on GitHub at [https://github.com/anycable/xk6-cable](https://github.com/anycable/xk6-cable).

## License

The gem is available as open source under the terms of the [MIT License](./LICENSE).

[k6]: https://k6.io
[xk6]: https://github.com/grafana/xk6
[k6-websockets]: https://k6.io/docs/using-k6/protocols/websockets/
