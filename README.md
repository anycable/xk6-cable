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

## Contributing

Bug reports and pull requests are welcome on GitHub at [https://github.com/anycable/xk6-cable](https://github.com/anycable/xk6-cable).

## License

The gem is available as open source under the terms of the [MIT License](./LICENSE).
