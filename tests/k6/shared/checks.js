/**
 * Check if the response is a JSON object and can be parsed.
 *
 * @return {boolean}
 */
export const isJson = (r) => {
  if (!r.headers['Content-Type'].includes('application/json')) {
    return false
  }

  try {
    JSON.parse(r.body)
  } catch (e) {
    return false
  }

  return true
}

/**
 * Checks if arrays are equal.
 *
 * @param {Array<*>} a
 * @param {Array<*>} b
 * @return {boolean}
 */
export const isArraysEqual = (a, b) => {
  return a.length === b.length && a.every((v, i) => {
    if (Array.isArray(v) && Array.isArray(b[i])) {
      return isArraysEqual(v, b)
    }

    if (typeof v === 'object' && typeof b[i] === 'object') {
      return deepEqual(v, b[i])
    }

    return v === b[i]
  })
}

/**
 * @param {Object} obj1
 * @param {Object} obj2
 * @return {boolean}
 */
function deepEqual(obj1, obj2) {
  if (obj1 === obj2) {
    return true
  }

  if (typeof obj1 !== 'object' || obj1 === null || typeof obj2 !== 'object' || obj2 === null) {
    return false
  }

  const keys1 = Object.keys(obj1)
  const keys2 = Object.keys(obj2)

  if (keys1.length !== keys2.length) {
    return false
  }

  for (const key of keys1) {
    if (!obj2.hasOwnProperty(key)) {
      return false
    }

    if (!deepEqual(obj1[key], obj2[key])) {
      return false
    }
  }

  return true
}

/**
 * Check if the string is a UUID and not the nil UUID.
 *
 * @param {String} uuid
 * @return {boolean}
 */
export const isUUID = (uuid) => {
  if (typeof uuid !== 'string') {
    return false
  }

  if (uuid.length !== 36) {
    return false
  }

  if (!uuid.match(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/)) {
    return false
  }

  return uuid !== '00000000-0000-0000-0000-000000000000';
}

/**
 * Check if the timestamp is within the delta.
 *
 * @param {Number} timestamp
 * @param {Number} delta
 * @return {boolean}
 */
export const isTimestampMillisInDelta = (timestamp, delta) => {
  return timestamp > new Date().getTime() - delta
}
