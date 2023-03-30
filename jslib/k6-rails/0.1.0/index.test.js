import { check } from 'k6'

import { turboStreamName, cableUrl, csrfToken, csrfParam, readMeta } from './index.js';

function testTurboStreamName() {
  const mockedData = {
    find: (_) => {
        return { attr: (_) => 'test name' }
    }
  };

  check(turboStreamName(mockedData), {
    'turboStreamName works': (r) => r === 'test name'
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
  testTurboStreamName,
  testCableUrl,
  testCsrfToken,
  testCsrfParam,
  testReadMeta,
}
