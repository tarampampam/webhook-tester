import React from 'react'
import { describe, expect, test, vi } from 'vitest'
import { act, render } from '@testing-library/react'
import { Client } from '~/api'
import { AppConfigProvider, useAppConfig } from './app-config'

type Config = Awaited<ReturnType<Client['getSettings']>>

const stubConfig: Config = Object.freeze({
  limits: Object.freeze({ maxRequests: 100, maxRequestBodySize: 65536, sessionTTL: 3600 }),
  tunnel: Object.freeze({ enabled: false, url: null }),
  publicUrlRoot: null,
})

const Consumer = (): React.JSX.Element => {
  const { config, error } = useAppConfig()
  return (
    <>
      <div data-testid="config">{JSON.stringify(config)}</div>
      <div data-testid="error">{error?.message ?? ''}</div>
    </>
  )
}

describe('AppConfigProvider', () => {
  test('fetches config and exposes it to children', async () => {
    const api = new Client({ baseUrl: 'http://test' })
    vi.spyOn(api, 'getSettings').mockResolvedValue(stubConfig)

    const { getByTestId } = render(
      <AppConfigProvider api={api}>
        <Consumer />
      </AppConfigProvider>
    )

    await act(async () => {})

    expect(getByTestId('config').textContent).toContain('"maxRequests":100')
    expect(getByTestId('error').textContent).toBe('')
  })

  test('exposes error when getSettings rejects', async () => {
    const api = new Client({ baseUrl: 'http://test' })
    vi.spyOn(api, 'getSettings').mockRejectedValue(new Error('network failure'))

    const { getByTestId } = render(
      <AppConfigProvider api={api}>
        <Consumer />
      </AppConfigProvider>
    )

    await act(async () => {})

    expect(getByTestId('config').textContent).toBe('null')
    expect(getByTestId('error').textContent).toBe('network failure')
  })
})
