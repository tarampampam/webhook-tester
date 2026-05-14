import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { afterEach, vi } from 'vitest'

dayjs.extend(relativeTime)

if (typeof window !== 'undefined') {
  window.HTMLElement.prototype.scrollIntoView = () => {}

  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })

  class ResizeObserver {
    observe = vi.fn()
    unobserve = vi.fn()
    disconnect = vi.fn()
  }

  window.ResizeObserver = ResizeObserver
}

// automatically unmount and cleanup DOM after the test is finished.
afterEach(() => cleanup())
