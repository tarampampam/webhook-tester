import http from 'k6/http'
import {b64decode, b64encode} from 'k6/encoding'
import {check, group} from './shared/k6.js'
import {randomString} from './shared/utils.js'
import {isArraysEqual, isJson, isTimestampMillisInDelta, isUUID} from './shared/checks.js'

/** @link https://grafana.com/docs/k6/latest/using-k6/k6-options/reference/ */
export const options = {
  scenarios: {
    default: {
      executor: 'per-vu-iterations',
      vus: 1, // force to use only one VU
    },
  },
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
  const {baseUrl} = ctx

  group('spa', () => {
    testSpaIndex(baseUrl)
    testSpaNotFound(baseUrl)
    testSpaRobots(baseUrl)
  })

  group('api', () => {
    testApiNotFound(baseUrl)
    testApiReady(baseUrl)
    testApiSettings(baseUrl)

    group('session', () => {
      testApiCreateSessionNegative(baseUrl)
      testApiSessionGetNegative(baseUrl)

      for (let statusCode = 200; statusCode <= 530; statusCode++) { // all possible status codes
        const [headerName, headerValue] = ['X-Custom' + randomString(10).toLowerCase(), randomString(10)]
        const responseBody = JSON.stringify({foo: randomString(10)})

        const sID = testApiCreateSession(baseUrl, statusCode, [{name: headerName, value: headerValue}], responseBody)

        testApiSessionGet(baseUrl, sID, {statusCode, headers: [{name: headerName, value: headerValue}], responseBody})

        group('requests', () => {
          testApiSessionHasNoRequests(baseUrl, sID) // initially, there are no requests
          testApiSessionCatchRequest(baseUrl, sID, {
            wantStatus: statusCode,
            wantHeaders: [{name: headerName, value: headerValue}],
            wantBody: responseBody,
          })
        })

        testApiSessionDelete(baseUrl, sID)
      }
    })
  })
}

/** @param {String} baseUrl */
const testSpaIndex = (baseUrl) => group('index', () =>
  check(http.get(baseUrl), {
    'status is 200': (r) => r.status === 200,
    'content type': (r) => r.headers['Content-Type'].includes('text/html'),
    'contains HTML': (r) => r.body.includes('<html'),
  })
)

/** @param {String} baseUrl */
const testSpaNotFound = (baseUrl) => group('404', () =>
  check(http.get(`${baseUrl}/foo${randomString(10)}`), {
    'status is 200': (r) => r.status === 200,
    'content type': (r) => r.headers['Content-Type'].includes('text/html'),
    'contains HTML': (r) => r.body.includes('<html'),
  })
)

/** @param {String} baseUrl */
const testSpaRobots = (baseUrl) => group('robots.txt', () =>
  check(http.get(`${baseUrl}/robots.txt`), {
    'status is 200': (r) => r.status === 200,
    'content type': (r) => r.headers['Content-Type'].includes('text/plain'),
    'contains useragent': (r) => r.body.includes('User-agent'),
    'contains disallow': (r) => r.body.includes('Disallow'),
  })
)

/** @param {String} baseUrl */
const testApiNotFound = (baseUrl) => group('404', () =>
  check(http.get(`${baseUrl}/////api/foo${randomString(10)}`), {
    'status is 404': (r) => r.status === 404,
    'is json': (r) => isJson(r),
    'contains error': (r) => r.body.includes('error'),
  })
)

/** @param {String} baseUrl */
const testApiReady = (baseUrl) => group('ready', () =>
  check(http.get(`${baseUrl}/ready`), {
    'status is 200': (r) => r.status === 200,
    'content type': (r) => r.headers['Content-Type'].includes('text/plain'),
    'contains ready': (r) => r.body.includes('OK'),
  })
)

/** @param {String} baseUrl */
const testApiSettings = (baseUrl) => group('settings', () =>
  check(http.get(`${baseUrl}/api/settings`), {
    'status is 200': (r) => r.status === 200,
    'is json': (r) => isJson(r),
    // TODO: add object properties checks
  })
)

/**
 * @param {String} baseUrl
 * @param {Number} statusCode
 * @param {Array<{name: String, value: String}>} headers
 * @param {String} respBody
 * @return {String} session ID
 */
const testApiCreateSession = (baseUrl, statusCode, headers, respBody) => group('create', () => {
  const resp = http.post(`${baseUrl}/api/session`, JSON.stringify({ // create a new session
    status_code: statusCode,
    headers: headers,
    delay: 0,
    response_body_base64: b64encode(respBody),
  }))

  check(resp, {
    'status is 200': (r) => r.status === 200,
    'is json': (r) => isJson(r),
    'uuid is not empty': (r) => isUUID(r.json('uuid')),
    'created_at is not empty': (r) => isTimestampMillisInDelta(r.json('created_at_unix_milli'), 2_000),
    'response status code is expected': (r) => r.json('response.status_code') === statusCode,
    'response headers': (r) => isArraysEqual(r.json('response.headers'), headers),
    'response delay': (r) => r.json('response.delay') === 0, // TODO: add test with delay
    'response response body (base64)': (r) => r.json('response.response_body_base64') === b64encode(respBody),
  })

  return resp.json('uuid')
})

/** @param {String} baseUrl */
const testApiCreateSessionNegative = (baseUrl) => group('create', () => {
  for (const [name, [wantErrSubstr, payload]] of Object.entries({
    'too small status code': ['wrong status code', {status_code: 99, headers: [], delay: 0, response_body_base64: ''}],
    'too big status code': ['wrong status code', {status_code: 531, headers: [], delay: 0, response_body_base64: ''}],
    'invalid header name': ['header key length', {
      status_code: 200,
      headers: [{name: '', value: 'bar'}],
      delay: 0,
      response_body_base64: ''
    }],
    'invalid header value': ['header value length', {
      status_code: 200,
      headers: [{name: 'foo', value: 'x'.repeat(2049)}],
      delay: 0,
      response_body_base64: ''
    }],
    'negative delay': ['delay', {status_code: 200, headers: [], delay: -1, response_body_base64: ''}],
    'too big delay': ['delay', {status_code: 200, headers: [], delay: 31, response_body_base64: ''}],
    'invalid base64': ['cannot decode response body', {
      status_code: 200,
      headers: [],
      delay: 0,
      response_body_base64: 'foobar'
    }],
  })) {
    group(name, () =>
      check(http.post(`${baseUrl}/api/session`, JSON.stringify(payload)), {
        'status is 400': (r) => r.status === 400,
        'is json': (r) => isJson(r),
        'contains error': (r) => r.body.includes(wantErrSubstr),
      })
    )
  }
})

/**
 * @param {String} baseUrl
 * @param {String} sID
 * @param {{statusCode: Number, headers: Array<{name: String, value: String}>, responseBody: string}} want
 */
const testApiSessionGet = (baseUrl, sID, want) => group('create', () =>
  check(http.get(`${baseUrl}/api/session/${sID}`), {
    'status is 200': (r) => r.status === 200,
    'is json': (r) => isJson(r),
    'uuid is not empty': (r) => isUUID(r.json('uuid')),
    'created_at is not empty': (r) => isTimestampMillisInDelta(r.json('created_at_unix_milli'), 2_000),
    'response status code': (r) => r.json('response.status_code') === want.statusCode,
    'response headers': (r) => isArraysEqual(r.json('response.headers'), want.headers),
    'response delay': (r) => r.json('response.delay') === 0,
    'response response body (base64)': (r) => r.json('response.response_body_base64') === b64encode(want.responseBody),
  })
)

/** @param {String} baseUrl */
const testApiSessionGetNegative = (baseUrl) => group('negative', () => {
  group('not found', () => {
    check(http.get(`${baseUrl}/api/session/00000000-0000-0000-0000-000000000000`), {
      'status is 404': (r) => r.status === 404,
      'is json': (r) => isJson(r),
      'contains error': (r) => r.body.includes('session not found'),
    })
  })

  group('wrong session ID format', () => {
    check(http.get(`${baseUrl}/api/session/foobar`), {
      'status is 400': (r) => r.status === 400,
      'is json': (r) => isJson(r),
      'contains error': (r) => r.body.includes('invalid UUID format'),
    })
  })
})

/**
 * @param {String} baseUrl
 * @param {String} sID
 */
const testApiSessionHasNoRequests = (baseUrl, sID) => group('requests', () =>
  check(http.get(`${baseUrl}/api/session/${sID}/requests`), {
    'status is 200': (r) => r.status === 200,
    'is json': (r) => isJson(r),
    'is array': (r) => Array.isArray(r.json()),
    'is empty': (r) => r.json().length === 0,
  })
)

/**
 * @param {String} baseUrl
 * @param {String} sID
 * @param {{wantStatus: Number, wantHeaders: Array<{name: String, value: String}>, wantBody: String}} want
 */
const testApiSessionCatchRequest = (baseUrl, sID, {wantStatus, wantHeaders, wantBody}) => group('catch', () => {
  for (const path of ['', '/foo', '////bar////////baz?yes=no']) {
    for (const payload of ['', 'foobar'.repeat(100)]) {
      for (const method of ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS']) {
        const [headerName, headerValue] = ['X-Custom' + randomString(10).toLowerCase(), randomString(10)]

        const resp = http.request(method, `${baseUrl}/${sID}${path}`, payload, { // send a request
          headers: {[headerName]: headerValue},
          tags: {sid: sID},
        })

        const responseMustBeEmpty = wantStatus === 204 || wantStatus === 304 // 204 = No Content, 304 = Not Modified
        const rID = resp.headers['X-Wh-Request-Id']

        check(resp, {
          'request id (from headers) is not empty': () => rID.length > 0,
          'response status': (r) => r.status === wantStatus,
          'response body is expected': (r) => r.body === (responseMustBeEmpty ? null : wantBody),
          'response headers are expected': (r) => wantHeaders.every(({name, value}) => r.headers[name] === value),
          'cors headers are expected': (r) => (
            r.headers['Access-Control-Allow-Origin'] === '*' &&
            r.headers['Access-Control-Allow-Methods'] === '*' &&
            r.headers['Access-Control-Allow-Headers'] === '*'
          ),
        })

        check(http.get(`${baseUrl}/api/session/${sID}/requests/${rID}`).json(), {
          'last request uuid': (r) => isUUID(r.uuid),
          'last request client address': (r) => r.client_address.length > 0,
          'last request method': (r) => r.method === method,
          'last request payload': (r) => b64decode(r.request_payload_base64, 'std', 's') === payload,
          'last request headers': (r) => r.headers.some(({
                                                           name,
                                                           value
                                                         }) => name === headerName && value === headerValue),
          'last request url': (r) => r.url.endsWith(`/${sID}${path}`),
        })
      }
    }
  }
})

/**
 * @param {String} baseUrl
 * @param {String} sID
 */
const testApiSessionDelete = (baseUrl, sID) => group('delete', () => {
  // ensure that the session exists first
  check(http.get(`${baseUrl}/api/session/${sID}`), {'status is 200': (r) => r.status === 200})

  // delete the session
  check(http.del(`${baseUrl}/api/session/${sID}`), {
    'status is 200': (r) => r.status === 200,
    'is json': (r) => isJson(r),
    'success': (r) => r.json('success') === true,
  })

  // try to get it back
  check(http.get(`${baseUrl}/api/session/${sID}`), {'status is 404': (r) => r.status === 404})

  // try to delete it again
  check(http.del(`${baseUrl}/api/session/${sID}`), {'status is 404': (r) => r.status === 404})
})
