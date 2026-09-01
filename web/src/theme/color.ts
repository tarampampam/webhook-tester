import type { MantineColor } from '@mantine/core'

/** Maps HTTP methods to specific colors for UI representation. */
export const methodToColor = (
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD' | 'OPTIONS' | 'CONNECT' | 'TRACE' | string
): MantineColor => {
  switch (method.trim().toUpperCase()) {
    case 'GET':
      return 'green'
    case 'POST':
      return 'yellow'
    case 'PUT':
      return 'blue'
    case 'PATCH':
      return 'violet'
    case 'DELETE':
      return 'red'
    case 'HEAD':
      return 'green'
    case 'OPTIONS':
      return 'orange'
    case 'TRACE':
      return 'pink'
    case 'CONNECT':
      return 'indigo'
  }

  return 'gray'
}

/** Maps HTTP status codes to specific colors for UI representation. */
export const statusCodeToColor = (statusCode: number): MantineColor => {
  switch (true) {
    case statusCode <= 299:
      return 'teal'
    case statusCode <= 399:
      return 'orange'
    case statusCode <= 499:
      return 'red'
    default:
      return 'cyan'
  }
}
