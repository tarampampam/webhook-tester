/**
 * Throws the given error if it is truthy.
 *
 * If the error is not an instance of Error, it is converted via {@link anyToError} before being thrown.
 *
 * **Falsy-value behavior** — the following values are treated as "no error" and cause the function to return silently:
 * - `null`, `undefined`
 * - empty string `""`
 * - `0`, `NaN`, `false`
 *
 * This means passing a numeric error code like `0` will **not** throw even though the caller may intend it to
 * represent an error condition.
 */
export const throwIfAny = (err: Error | string | unknown): void => {
  if (err) {
    throw anyToError(err)
  }
}

/**
 * Converts any unknown value to an `Error` instance.
 *
 * The conversion rules are applied in the following priority order:
 *
 * | Input type / shape                           | Result                           |
 * |----------------------------------------------|----------------------------------|
 * | `Error` (or subclass)                        | returned as-is (identity)        |
 * | `string`                                     | `new Error(str)`                 |
 * | `null` or `undefined`                        | `new Error('Unknown error')`     |
 * | object with `message: string`                | `new Error(obj.message)`         |
 * | object with `toString()`                     | `new Error(obj.toString())`      |
 * | object with `cause: string`                  | `new Error(obj.cause)`           |
 * | object with `cause: Error`                   | `obj.cause` returned as-is       |
 * | any other object                             | `new Error(JSON.stringify(obj))` |
 * | any other primitive (`number`, `boolean`, …) | `new Error(String(value))`       |
 *
 */
export const anyToError = (err: unknown): Error => {
  switch (true) {
    case err instanceof Error:
      return err
    case typeof err === 'string':
      return new Error(err)
    case err === null || err === undefined:
      return new Error('Unknown error')
    case typeof err === 'object':
      switch (true) {
        case 'message' in err && typeof err.message === 'string':
          return new Error(err.message)
        case 'toString' in err && typeof err.toString === 'function':
          return new Error(err.toString())
        case 'cause' in err && typeof err.cause === 'string':
          return new Error(err.cause)
        case 'cause' in err && err.cause instanceof Error:
          return err.cause
      }

      return new Error(JSON.stringify(err))
  }

  return new Error(String(err))
}
