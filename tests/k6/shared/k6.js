import {check as k6check, group as k6group} from 'k6'
import execution from 'k6/execution'

/**
 * This is an overridden version of the native check function, designed to abort the test when an expectation fails.
 * In addition, it improves the type checking by using the JSDoc annotation.
 *
 * @template VT
 * @param {VT} val
 * @param {Record<string, (VT) => Boolean>} set
 * @returns {boolean}
 */
export const check = (val, set) => {
  try {
    for (const [key, fn] of Object.entries(set)) {
      if (!k6check(val, { [key]: fn })) {
        execution.test.abort(`Failed expectation: ${key}`)
      }
    }
  } catch (e) {
    execution.test.abort(String(e))

    throw e
  }
}

/**
 * This is an overridden version of the native group function, designed to abort the test when an error occurs.
 *
 * @template RT
 * @param {String} name
 * @param {() => RT} fn
 * @returns {RT}
 */
export const group = (name, fn) => {
  try {
    return k6group(name, fn)
  } catch (e) {
    execution.test.abort(String(e))

    throw e
  }
}
