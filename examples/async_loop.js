// Build k6 with xk6-cable like this:
//    xk6 build v0.38.3 --with github.com/anycable/xk6-cable@v0.3.0

import { check, sleep, fail } from "k6";
import cable from "k6/x/cable";
import { randomIntBetween } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { Trend, Counter } from "k6/metrics";
let rttTrend = new Trend("rtt", true);
let broadcastsRcvd = new Counter("broadcasts_rcvd");
let broadcastsSent = new Counter("broadcasts_sent");

let config = __ENV

config.URL = config.URL || "ws://localhost:8080/cable";

let url = config.URL;
let channelName = 'BenchmarkChannel';

export default function () {
  let cableOptions = {
    receiveTimeoutMs: 15000
  }

  let client = cable.connect(url, cableOptions);

  if (
    !check(client, {
      "successful connection": (obj) => obj,
    })
  ) {
    fail("connection failed");
  }

  let channel = client.subscribe(channelName);

  if (
    !check(channel, {
      "successful subscription": (obj) => obj,
    })
  ) {
    fail("failed to subscribe");
  }

  channel.ignoreReads();

  channel.onMessage(data => {
    let now = Date.now();
    let { message } = data;

    if (!message) {
      console.log(data);
      return
    }

    if (message.action == "broadcast") {
      broadcastsRcvd.add(1);
      let ts = message.ts;
      rttTrend.add(now - ts);
    }
  })

  let i = 0;

  client.loop(() => {
    i++;
    // Sampling
    if (randomIntBetween(1, 10) > 8) {
      let start = Date.now();
      broadcastsSent.add(1);
      // Create message via cable instead of a form
      channel.perform("broadcast", { ts: start, content: `hello from ${__VU} numero ${i+1}` });
    }

    sleep(randomIntBetween(5, 10) / 100);
  })
}
