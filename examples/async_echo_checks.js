import { check } from "k6";
import cable from "k6/x/cable";

export default function () {
  const client = cable.connect("ws://localhost:8080/cable");
  const channel = client.subscribe("AsyncEchoChannel");

  // Bind a callback for all channel's input messages
  channel.onMessage(msg => {
    check(msg, {
      "received command": obj => obj.speak === "hello",
    });
  });

  // Successful check
  channel.perform("echo", { speak: "hello" });
  channel.receive();

  // Failed check
  channel.perform("echo", { foo: 1 });
  channel.receive();

  // Sync check passed but async failed
  channel.perform("echo", { foobar: 3 });
  channel.perform("echo", { foobaz: 3 });
  channel.perform("echo", { baz: 3 });
  const reses = channel.receiveN(3);
  check(reses, {
    "received 3 messages": obj => obj.length === 3,
  });

  // Redefine callback
  channel.onMessage(msg => {
    check(msg, {
      "received phrase": obj => obj.test === 1,
    });
  });
  channel.perform("echo", { test: 1 });
  channel.receive();
}
