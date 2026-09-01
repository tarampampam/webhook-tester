import { describe, expect, test } from 'vitest'
import { render } from '~/test-utils/render'
import { LogoText } from './logo-text'

describe('LogoText', () => {
  test('renders without crashing', () => {
    const { container } = render(<LogoText />)
    expect(container.firstChild).not.toBeNull()
  })

  test('renders an svg element', () => {
    const { container } = render(<LogoText />)
    expect(container.querySelector('svg')).not.toBeNull()
  })

  test('forwards extra props to the svg element', () => {
    const { container } = render(<LogoText data-testid="logo" />)
    expect(container.querySelector('[data-testid="logo"]')).not.toBeNull()
  })
})
