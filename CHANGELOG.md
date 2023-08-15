# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog],
and this project adheres to [Semantic Versioning].

## [Unreleased]

## [0.7.0]

- Add `channel.ackDuration()` to get the number of milliseconds to wait for a subscription confirmation/rejection. ([@palkan][])

- Add `client.subscribeAsync` to issue a `subscribe` command without waiting for the confirmation. ([@palkan][])

- Fix `k6` / Logrus compatibility issue. ([@palkan][])

## [0.6.0]

- Add JS helpers.

```js
import { cableUrl, turboStreamSource } from 'https://anycable.io/xk6-cable/jslib/k6-rails/0.1.0/index.js'

export default function () {
  let res = http.get("http://localhost:3000/home");
  const html = res.html();

  const wsUrl = cableUrl(html);
  let client = cable.connect(wsUrl);

  let { streamName, channelName } = turboStreamSource(html);

  let channel = client.subscribe(channelName, {
    signed_stream_name: streamName,
  });

  // ...
}
```

## [0.5.0]

- Add `__timestamp__` field to incoming messages with the receive time (as UTC milliseconds). ([@palkan][])

This should be used to determine the actual time when the message was received (not when it reached JS runtime).

- Add `client.Loop`. ([@palkan][])

This makes it possible to use shared data along with `onMessage` callbacks.
Without wrapping code into `client.Loop`, JS runtime race conditions could occur.

## [0.4.0]

- Add `channel.OnMessage` to process incoming messages asynchronously. ([@SlayerDF][])

## [0.3.0] - 2022-06-23

### Changed

- Update k6 dependency to latest v0.38.3 release. ([@skryukov])

## [0.2.0] - 2021-11-29

### Changed

- Update k6 dependency to latest v0.35.0 release. ([@skryukov])

## [0.1.0] - 2021-09-15

### Added

- Protobuf codec support. ([@palkan])

### Fixed

- Fix setting logging level. ([@palkan])

## [0.0.3] - 2021-09-12

### Added

- Ability to change logging level with `logLevel` option. ([@palkan])

## [0.0.2] - 2021-09-10

### Added

- New `channel.ignoreReads()` method, that allows skipping collecting incoming messages. ([@palkan])

- More examples. ([@palkan])

### Changed

- Use buffered channels to receive messages. ([@palkan])

## [0.0.1] - 2021-08-19

### Added

- Initial implementation. ([@skryukov], [@palkan])

[@skryukov]: https://github.com/skryukov
[@palkan]: https://github.com/palkan
[@SlayerDF]: https://github.com/SlayerDF

[Unreleased]: https://github.com/anycable/xk6-cable/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/anycable/xk6-cable/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/anycable/xk6-cable/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/anycable/xk6-cable/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/anycable/xk6-cable/compare/v0.0.3...v0.1.0
[0.0.3]: https://github.com/anycable/xk6-cable/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/anycable/xk6-cable/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/anycable/xk6-cable/releases/tag/v0.0.1

[Keep a Changelog]: https://keepachangelog.com/en/1.0.0/
[Semantic Versioning]: https://semver.org/spec/v2.0.0.html
