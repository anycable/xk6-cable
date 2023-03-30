// This example is extracted from AnyCable demo app: https://github.com/anycable/anycable_rails_demo/blob/feat/hotwire-k6/etc/k6/chat.js

import { check, sleep, fail } from "k6";
import http from "k6/http";
import cable from "k6/x/cable";
import { randomIntBetween } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { cableUrl, turboStreamName } from 'http://anycable.io/xk6-cable/jslib/k6-rails/0.1.0/index.js'

import { Trend } from "k6/metrics";

let rttTrend = new Trend("rtt", true);

let userId = `100${__VU}`;
let userName = `Kay${userId}`;

export default function () {
  // Manually set authentication cookies
  let jar = http.cookieJar();
  jar.set("http://localhost:3000", "uid", `${userName}/${userId}`);

  let res = http.get("http://localhost:3000/workspaces/demo");

  if (
    !check(res, {
      "is status 200": (r) => r.status === 200,
    })
  ) {
    fail("couldn't open dashboard");
  }

  const html = res.html();
  const wsUrl = cableUrl(html);

  if (!wsUrl) {
    fail("couldn't find cable url on the page");
  }

  let client = cable.connect(wsUrl, {
    cookies: `uid=${userName}/${userId}`,
  });

  if (
    !check(client, {
      "successful connection": (obj) => obj,
    })
  ) {
    fail("connection failed");
  }

  let streamName = turboStreamName(html);

  if (!streamName) {
    fail("couldn't find a turbo stream element");
  }

  let channel = client.subscribe("Turbo::StreamsChannel", {
    signed_stream_name: streamName,
  });

  if (
    !check(channel, {
      "successful subscription": (obj) => obj,
    })
  ) {
    fail("failed to subscribe");
  }

  for (let i = 0; i < 5; i++) {
    let startMessage = Date.now();

    // We have an HTML form to submit chat messages,
    // submitting it initiates a broadcasting
    let formRes = res.submitForm({
      formSelector: ".chat form",
      fields: { message: `hello from ${userName}` },
    });

    if (
      !check(formRes, {
        "is status 200": (r) => r.status === 200,
      })
    ) {
      fail("couldn't submit message form");
    }

    // Msg here is an HTML element (<turbo-stream>),
    // we use data attributes to indicate the message author,
    // so, here we're looking for our messages
    let message = channel.receive((msg) => {
      return msg.includes(`data-author-id="${userId}"`);
    });

    if (
      !check(message, {
        "received its own message": (obj) => obj,
      })
    ) {
      fail("expected message hasn't been received");
    }

    let endMessage = Date.now();
    rttTrend.add(endMessage - startMessage);

    sleep(randomIntBetween(5, 10) / 10);
  }

  client.disconnect();
}
