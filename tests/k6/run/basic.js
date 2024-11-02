import execution from 'k6/execution'
import http from 'k6/http'
import {group, check as k6check} from 'k6'
import {randomString} from '../shared/utils.js'

/** @link https://grafana.com/docs/k6/latest/using-k6/k6-options/reference/ */
export const options = {
  scenarios: {
    default: {
      executor: 'per-vu-iterations',
      vus: 1, // force to use only one VU
    },
  },
}

/**
 * This is an overridden version of the native check function, designed to abort the test when an expectation fails.
 * In addition, it improves the type checking by using the JSDoc annotation.
 *
 * @template VT
 * @param {VT} val
 * @param {Record<string, (VT) => Boolean>} set
 */
const check = (val, set) => {
  try {
    if (!k6check(val, set)) {
      execution.test.abort('Failed expectation: ' + Object.keys(set).join(', '))
    }
  } catch (e) {
    execution.test.abort(String(e))

    throw e
  }
}

/** @typedef {{baseUrl: String}} Context */
/** @return Context */
export const setup = () => {
  const baseUrl = __ENV['BASE_URL']

  if (!baseUrl) {
    throw new Error('BASE_URL is required')
  }

  return {
    baseUrl: baseUrl.replace(/\/$/, ''), // remove trailing slash
  }
}

/** @param {Context} ctx */
export default (ctx) => {
  group('index', () => {
    const resp = http.get(ctx.baseUrl)

    check(resp, {
      'status is 200': (r) => r.status === 200,
      'content type': (r) => r.headers['Content-Type'].includes('text/html'),
      'contains HTML': (r) => r.body.includes('<html'),
    })
  })

  group('robots.txt', () => {
    const resp = http.get(`${ctx.baseUrl}/robots.txt`)

    check(resp, {
      'status is 200': (r) => r.status === 200,
      'content type': (r) => r.headers['Content-Type'].includes('text/plain'),
      'contains useragent': (r) => r.body.includes('User-agent'),
      'contains disallow': (r) => r.body.includes('Disallow'),
    })
  })

  group('spa 404', () => {
    const resp = http.get(`${ctx.baseUrl}/foo${randomString(10)}`)

    check(resp, {
      'status is 200': (r) => r.status === 200,
      'content type': (r) => r.headers['Content-Type'].includes('text/html'),
      'contains HTML': (r) => r.body.includes('<html'),
    })
  })

  group('api 404', () => {
    const resp = http.get(`${ctx.baseUrl}/////api/foo${randomString(10)}`)

    check(resp, {
      'status is 404': (r) => r.status === 404,
      'content type': (r) => r.headers['Content-Type'].includes('application/json'),
      'contains error': (r) => r.body.includes('error'),
    })
  })

  group('ready handler', () => { // outside the /api path
    const resp = http.get(`${ctx.baseUrl}/ready`)

    check(resp, {
      'status is 200': (r) => r.status === 200,
      'content type': (r) => r.headers['Content-Type'].includes('text/plain'),
      'contains ready': (r) => r.body.includes('OK'),
    })
  })
}
