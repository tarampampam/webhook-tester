import React from 'react'
import { describe, expect, test } from 'vitest'
import { render, renderHook } from '@testing-library/react'
import { WebhookURLProvider, useWebhookURL } from './webhook-url'

const makeWrapper = (sID: string | null, publicUrlRoot: URL | null) =>
  function Wrapper({ children }: { children: React.ReactNode }): React.JSX.Element {
    return (
      <WebhookURLProvider sID={sID} publicUrlRoot={publicUrlRoot}>
        {children}
      </WebhookURLProvider>
    )
  }

const URLConsumer = (): React.JSX.Element => {
  const { webhookURL } = useWebhookURL()
  return <div data-testid="url">{webhookURL?.toString() ?? 'null'}</div>
}

describe('useWebhookURL', () => {
  test('throws when called outside WebhookURLProvider', () => {
    expect(() => renderHook(() => useWebhookURL())).toThrow(
      'useWebhookURL must be used within a WebhookURLProvider'
    )
  })
})

describe('WebhookURLProvider', () => {
  test('webhookURL is null when sID is null', () => {
    const { result } = renderHook(() => useWebhookURL(), {
      wrapper: makeWrapper(null, new URL('http://example.com')),
    })

    expect(result.current.webhookURL).toBeNull()
  })

  test('webhookURL is null when sID is an empty string', () => {
    const { result } = renderHook(() => useWebhookURL(), {
      wrapper: makeWrapper('', new URL('http://example.com')),
    })

    expect(result.current.webhookURL).toBeNull()
  })

  test('builds URL from publicUrlRoot and sID', () => {
    const { result } = renderHook(() => useWebhookURL(), {
      wrapper: makeWrapper('abc-123', new URL('http://example.com')),
    })

    expect(result.current.webhookURL?.toString()).toBe('http://example.com/abc-123')
  })

  test('appends sID as child path segment when publicUrlRoot has a subpath', () => {
    const { result } = renderHook(() => useWebhookURL(), {
      wrapper: makeWrapper('abc-123', new URL('http://example.com/webhooks')),
    })

    expect(result.current.webhookURL?.toString()).toBe('http://example.com/webhooks/abc-123')
  })

  test('handles trailing slash in publicUrlRoot', () => {
    const { result } = renderHook(() => useWebhookURL(), {
      wrapper: makeWrapper('abc-123', new URL('http://example.com/webhooks/')),
    })

    expect(result.current.webhookURL?.toString()).toBe('http://example.com/webhooks/abc-123')
  })

  test('falls back to window.location.origin when publicUrlRoot is null', () => {
    const { result } = renderHook(() => useWebhookURL(), {
      wrapper: makeWrapper('abc-123', null),
    })

    expect(result.current.webhookURL?.toString()).toBe(`${window.location.origin}/abc-123`)
  })

  test('returned URL is frozen', () => {
    const { result } = renderHook(() => useWebhookURL(), {
      wrapper: makeWrapper('abc-123', new URL('http://example.com')),
    })

    expect(Object.isFrozen(result.current.webhookURL)).toBe(true)
  })

  test('webhookURL updates when sID changes', () => {
    const { getByTestId, rerender } = render(
      <WebhookURLProvider sID="session-1" publicUrlRoot={new URL('http://example.com')}>
        <URLConsumer />
      </WebhookURLProvider>
    )

    expect(getByTestId('url').textContent).toBe('http://example.com/session-1')

    rerender(
      <WebhookURLProvider sID="session-2" publicUrlRoot={new URL('http://example.com')}>
        <URLConsumer />
      </WebhookURLProvider>
    )

    expect(getByTestId('url').textContent).toBe('http://example.com/session-2')
  })

  test('webhookURL becomes null when sID changes to null', () => {
    const { getByTestId, rerender } = render(
      <WebhookURLProvider sID="session-1" publicUrlRoot={new URL('http://example.com')}>
        <URLConsumer />
      </WebhookURLProvider>
    )

    rerender(
      <WebhookURLProvider sID={null} publicUrlRoot={new URL('http://example.com')}>
        <URLConsumer />
      </WebhookURLProvider>
    )

    expect(getByTestId('url').textContent).toBe('null')
  })
})
