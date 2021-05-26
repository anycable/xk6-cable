import cable from "k6/x/cable";

export default function () {
    const client = cable.connect("ws://localhost:8080/cable");
    const channel = client.subscribe("BenchmarkChannel");
    console.log('channel', channel)

    channel.perform('echo', {foo: 1});
    const res = channel.receive();
    console.log('res', JSON.stringify(res))

    channel.perform('echo', {bar: 2});
    const res2 = channel.receive((msg) => msg['bar'] === 2);
    console.log('res2', JSON.stringify(res2))

    channel.perform('echo', {foobar: 3});
    channel.perform('echo', {foobaz: 3});
    channel.perform('echo', {baz: 3});
    const res3 = channel.receive({baz: 3});
    console.log('res3', JSON.stringify(res3))

    channel.perform('echo', {foobar: 3});
    channel.perform('echo', {foobaz: 3});
    channel.perform('echo', {baz: 3});
    const reses = channel.receiveN(3);
    console.log('reses', JSON.stringify(reses))

    channel.perform('echo', {baz: 3});
    channel.perform('echo', {foobaz: 3});
    channel.perform('echo', {baz: 3, foobaz: 3});
    const reses2 = channel.receiveN(2, {action: 'echo', baz: 3});
    console.log('reses2', JSON.stringify(reses2))

    channel.perform('echo', {baz: 3});
    channel.perform('echo', {foobaz: 3});
    channel.perform('echo', {baz: 3});
    const reses3 = channel.receiveN(2, (msg) => msg['action'] === 'echo');
    console.log('reses3', JSON.stringify(reses3))
}
