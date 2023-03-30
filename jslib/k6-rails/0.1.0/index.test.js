import { check } from 'k6'

import { turboStreamSource, cableUrl, csrfToken, csrfParam, readMeta } from './index.js';

function testTurboStreamSource() {
  const mockedData = {
    find: (_) => {
        return {
          attr: (name) => {
            if (name === 'signed-stream-name') return 'test name'
          }
        }
    }
  };

  check(turboStreamSource(mockedData), {
    'turboStreamSource works': (r) => r.streamName === 'test name' && r.channelName === 'Turbo::StreamsChannel'
  })
}

function testCableUrl() {
  const mockedData = {
    find: (_) => {
      return { attr: (_) => 'cable url' }
    }
  }

  check(cableUrl(mockedData), {
    'cableUrl works': (r) => r === 'cable url'
  })
}

function testCsrfToken() {
  const mockedData = {
    find: (_) => {
      return { attr: (_) => 'csrf-token' }
    }
  }

  check(csrfToken(mockedData), {
    'csrfToken works': (r) =>  r === 'csrf-token'
  })
}

function testCsrfParam() {
  const mockedData = {
    find: (_) => {
      return { attr: (_) => 'csrf-param' }
    }
  }

  check(csrfParam(mockedData), {
    'csrfToken works': (r) =>  r === 'csrf-param'
  })
}

function testReadMeta() {
  const mockedData = {
    find: (_) => {
      return { attr: (_) => 'width=device-width, initial-scale=1' }
    }
  }

  check(readMeta(mockedData, 'name', 'viewport', 'content'), {
    'readMeta works': (r) => r === 'width=device-width, initial-scale=1'
  })
}

export {
  testTurboStreamSource,
  testCableUrl,
  testCsrfToken,
  testCsrfParam,
  testReadMeta,
}
