# xk6-cable

A k6 extension for testing Action Cable and AnyCable functionality. Built for [k6](https://go.k6.io/k6) using the [xk6](https://github.com/k6io/xk6) system.

## Build

To build a `k6` binary with this extension, first ensure you have the prerequisites:

- [Go toolchain](https://go101.org/article/go-toolchain.html)
- Git

1. Install `xk6` framework for extending `k6`:
```shell
go install github.com/k6io/xk6/cmd/xk6@latest
```

2. Build the binary:
```shell
xk6 build --with github.com/anycable/xk6-cable@latest
```

## Example
```shell
./k6 run example.js
```

Result output:
```shell

          /\      |‾‾| /‾‾/   /‾‾/
     /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
   /          \   |  |\  \ |  (‾)  |
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: example.js
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

## Contributing

Bug reports and pull requests are welcome on GitHub at [https://github.com/anycable/xk6-cable](https://github.com/anycable/xk6-cable).

## License

The gem is available as open source under the terms of the [MIT License](./LICENSE).
