import { describe, expect, test } from 'vitest'
import { render } from '~/test-utils/render'
import { Logo } from './logo'

describe('Logo', () => {
  test('renders without crashing', () => {
    const { container } = render(<Logo />)
    expect(container.firstChild).not.toBeNull()
  })

  test('renders an svg element', () => {
    const { container } = render(<Logo />)
    expect(container.querySelector('svg')).not.toBeNull()
  })

  test('forwards extra props to the svg element', () => {
    const { container } = render(<Logo data-testid="logo-icon" />)
    expect(container.querySelector('[data-testid="logo-icon"]')).not.toBeNull()
  })
})
