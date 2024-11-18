import type { MantineColor } from '@mantine/core'

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
