import { describe, expect, test } from 'vitest'
import { render } from '~/test-utils/render'
import { ImagePanda } from './image-panda'

describe('ImagePanda', () => {
  test('renders without crashing', () => {
    const { container } = render(<ImagePanda />)
    expect(container.firstChild).not.toBeNull()
  })

  test('renders an svg element', () => {
    const { container } = render(<ImagePanda />)
    expect(container.querySelector('svg')).not.toBeNull()
  })

  test('forwards extra props to the svg element', () => {
    const { container } = render(<ImagePanda data-testid="panda" />)
    expect(container.querySelector('[data-testid="panda"]')).not.toBeNull()
  })
})
