/**
 * Returns a random integer between min (inclusive) and max (inclusive).
 *
 * @param {Number} min The minimum value
 * @param {Number} max The maximum value
 * @returns {Number} The random integer
 */
export const randomIntBetween = (min, max) => Math.floor(Math.random() * (max - min + 1) + min)

/**
 * Returns a random element from the array.
 *
 * @template T
 * @param {Array<T>} array The array
 * @returns {T} The random element
 */
export const randomArrayElement = (array) => array[randomIntBetween(0, array.length - 1)]

/**
 * Generates a random string of the specified length.
 *
 * @param {Number} length The length of the string
 * @param {String} [charset] The charset to use
 * @returns {String} The random string
 */
export const randomString = (length, charset) => {
  let res = ''

  if (!charset) {
    charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
  }

  while (length--) {
    res += charset[(Math.random() * charset.length) | 0]
  }

  return res
}
