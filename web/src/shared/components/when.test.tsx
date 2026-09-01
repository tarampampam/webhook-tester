import { describe, expect, test } from 'vitest'
import { render } from '~/test-utils/render'
import { When } from './when'

describe('When', () => {
  test('applies wrapper when condition is true', () => {
    const { container } = render(
      <When condition={true} wrapper={(children) => <span data-testid="wrapper">{children}</span>}>
        content
      </When>
    )

    expect(container.querySelector('[data-testid="wrapper"]')).not.toBeNull()
    expect(container.textContent).toContain('content')
  })

  test('renders children directly when condition is false', () => {
    const { container } = render(
      <When condition={false} wrapper={(children) => <span data-testid="wrapper">{children}</span>}>
        content
      </When>
    )

    expect(container.querySelector('[data-testid="wrapper"]')).toBeNull()
    expect(container.textContent).toContain('content')
  })
})
