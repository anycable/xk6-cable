import { check } from "k6";
import cable from "k6/x/cable";

export default function () {
  const client = cable.connect("ws://localhost:8080/cable");
  const channel = client.subscribe("BenchmarkChannel");

  channel.perform("echo", { foo: 1 });
  const res = channel.receive();
  check(res, {
    "received res": (obj) => obj.foo === 1,
  });

  channel.perform("echo", { bar: 2 });
  const res2 = channel.receive((msg) => msg.bar === 2);
  check(res2, {
    "received res2": (obj) => obj.bar === 2,
  });

  channel.perform("echo", { foobar: 3 });
  channel.perform("echo", { foobaz: 3 });
  channel.perform("echo", { baz: 3 });
  const reses = channel.receiveN(3);
  check(reses, {
    "received 3 messages": (obj) => obj.length === 3,
  });

  channel.perform("echo", { baz: 3 });
  channel.perform("echo", { foobaz: 3 });
  channel.perform("echo", { baz: 3, foobaz: 3 });
  const reses2 = channel.receiveN(2, { baz: 3 });
  check(reses2, {
    "received 2 messages": (obj) => obj.length === 2,
    "all messages with baz attr": (obj) =>
      obj.reduce((r, e) => r && e.baz === 3, true),
  });
}
