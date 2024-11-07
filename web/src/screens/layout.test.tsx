import { MantineProvider } from '@mantine/core'
import { SemVer } from 'semver'
import { describe, afterEach, expect, test, vi } from 'vitest'
import { render, cleanup } from '~/test-utils'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { default as Layout } from './layout'
import { Client } from '~/api'

afterEach(() => cleanup())

/** @jest-environment jsdom */
describe('layout', () => {
  test('common', () => {
    const apiClient = new Client({ baseUrl: 'http://unit/test' })
    const FakeComponent = () => <div>fake text</div>

    vi.spyOn(apiClient, 'currentVersion').mockResolvedValueOnce(new SemVer('0.0.0-unit-test'))
    vi.spyOn(apiClient, 'latestVersion').mockResolvedValueOnce(new SemVer('99.0.0-unit-test'))
    vi.spyOn(apiClient, 'getSettings').mockResolvedValueOnce({
      limits: { maxRequests: 1000, sessionTTL: 60, maxRequestBodySize: 1000 },
      tunnel: { enabled: true, url: new URL('http://unit/test') },
    })

    const { unmount } = render(
      <MantineProvider>
        <MemoryRouter initialEntries={['/']}>
          <Routes>
            <Route element={<Layout apiClient={apiClient} />}>
              <Route path="/" element={<FakeComponent />} />
            </Route>
          </Routes>
        </MemoryRouter>
      </MantineProvider>
    )

    expect(apiClient.currentVersion).toHaveBeenCalledTimes(1)
    expect(apiClient.latestVersion).toHaveBeenCalledTimes(1)

    unmount()
  })
})
