import { Rate } from 'k6/metrics'

import {
  testTurboStreamSource,
  testCableUrl,
  testCsrfToken,
  testCsrfParam,
  testReadMeta
} from './k6-rails/0.1.0/index.test.js'

let testCasesOK = new Rate('test_case_ok')

const testCases = [
  testTurboStreamSource,
  testCableUrl,
  testCsrfToken,
  testCsrfParam,
  testReadMeta,
]

export const options = {
  vus: 1,
  iterations: testCases.length,
  thresholds: {
    checks: ['rate==1.0'],
    test_case_ok: ['rate==1.0'],
  },
}

export default function () {
  try {
    testCases[__ITER]()
    testCasesOK.add(true)
  } catch (e) {
    testCasesOK.add(false)
    console.log(`test case at index ${__ITER} has failed`)
    throw e
  }
}
