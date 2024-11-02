import type { MantineColor } from '@mantine/core'

export const methodToColor = (
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD' | 'OPTIONS' | 'CONNECT' | 'TRACE' | string
): MantineColor => {
  switch (method.trim().toUpperCase()) {
    case 'GET':
      return 'blue'
    case 'POST':
      return 'green'
    case 'PUT':
      return 'yellow'
    case 'PATCH':
      return 'purple'
    case 'DELETE':
      return 'red'
    case 'HEAD':
      return 'teal'
    case 'OPTIONS':
      return 'orange'
    case 'TRACE':
      return 'pink'
    case 'CONNECT':
      return 'indigo'
  }

  return 'gray'
}
