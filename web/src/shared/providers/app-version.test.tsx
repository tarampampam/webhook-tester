import React from 'react'
import { describe, expect, test, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { Client } from '~/api'
import { AppVersionProvider, useAppVersion } from './app-version'

const makeWrapper = (api: Client) =>
  function Wrapper({ children }: { children: React.ReactNode }): React.JSX.Element {
    return <AppVersionProvider api={api}>{children}</AppVersionProvider>
  }

const pending = (): Promise<Readonly<string>> => new Promise<Readonly<string>>(() => {})

describe('useAppVersion', () => {
  test('throws when called outside AppVersionProvider', () => {
    expect(() => renderHook(() => useAppVersion())).toThrow('useAppVersion must be used within an AppVersionProvider')
  })
})

describe('AppVersionProvider', () => {
  test('sets current after currentVersion resolves', async () => {
    const api = new Client({ baseUrl: 'http://test' })
    vi.spyOn(api, 'currentVersion').mockResolvedValue('1.2.3')
    vi.spyOn(api, 'latestVersion').mockReturnValue(pending())

    const { result } = renderHook(() => useAppVersion(), { wrapper: makeWrapper(api) })

    await waitFor(() => expect(result.current.current).toBe('1.2.3'))
    expect(result.current.latest).toBeNull()
  })

  test('sets latest after latestVersion resolves', async () => {
    const api = new Client({ baseUrl: 'http://test' })
    vi.spyOn(api, 'currentVersion').mockReturnValue(pending())
    vi.spyOn(api, 'latestVersion').mockResolvedValue('2.0.0')

    const { result } = renderHook(() => useAppVersion(), { wrapper: makeWrapper(api) })

    await waitFor(() => expect(result.current.latest).toBe('2.0.0'))
    expect(result.current.current).toBeNull()
  })

  test('updateAvailable is null when only one version is known', async () => {
    const api = new Client({ baseUrl: 'http://test' })
    vi.spyOn(api, 'currentVersion').mockResolvedValue('1.0.0')
    vi.spyOn(api, 'latestVersion').mockReturnValue(pending())

    const { result } = renderHook(() => useAppVersion(), { wrapper: makeWrapper(api) })

    await waitFor(() => expect(result.current.current).toBe('1.0.0'))
    expect(result.current.updateAvailable).toBeNull()
  })

  test('updateAvailable is true when latest is newer', async () => {
    const api = new Client({ baseUrl: 'http://test' })
    vi.spyOn(api, 'currentVersion').mockResolvedValue('1.0.0')
    vi.spyOn(api, 'latestVersion').mockResolvedValue('1.0.1')

    const { result } = renderHook(() => useAppVersion(), { wrapper: makeWrapper(api) })

    await waitFor(() => expect(result.current.updateAvailable).toBe(true))
  })

  test('updateAvailable is false when versions are equal', async () => {
    const api = new Client({ baseUrl: 'http://test' })
    vi.spyOn(api, 'currentVersion').mockResolvedValue('1.0.0')
    vi.spyOn(api, 'latestVersion').mockResolvedValue('1.0.0')

    const { result } = renderHook(() => useAppVersion(), { wrapper: makeWrapper(api) })

    await waitFor(() => expect(result.current.updateAvailable).toBe(false))
  })

  test('updateAvailable handles pre-release suffix in version strings', async () => {
    const api = new Client({ baseUrl: 'http://test' })
    vi.spyOn(api, 'currentVersion').mockResolvedValue('1.0.3-beta')
    vi.spyOn(api, 'latestVersion').mockResolvedValue('1.0.3')

    const { result } = renderHook(() => useAppVersion(), { wrapper: makeWrapper(api) })

    // parseInt("3-beta", 10) parses as 3 - same as "1.0.3" patch, so not newer
    await waitFor(() => expect(result.current.updateAvailable).toBe(false))
  })

  test('sets error when currentVersion fetch fails', async () => {
    const api = new Client({ baseUrl: 'http://test' })
    vi.spyOn(api, 'currentVersion').mockRejectedValue(new Error('current unavailable'))
    vi.spyOn(api, 'latestVersion').mockReturnValue(pending())

    const { result } = renderHook(() => useAppVersion(), { wrapper: makeWrapper(api) })

    await waitFor(() => expect(result.current.error).not.toBeNull())
    expect(result.current.error?.message).toBe('current unavailable')
    expect(result.current.current).toBeNull()
  })

  test('sets error when latestVersion fetch fails', async () => {
    const api = new Client({ baseUrl: 'http://test' })
    vi.spyOn(api, 'currentVersion').mockReturnValue(pending())
    vi.spyOn(api, 'latestVersion').mockRejectedValue(new Error('latest unavailable'))

    const { result } = renderHook(() => useAppVersion(), { wrapper: makeWrapper(api) })

    await waitFor(() => expect(result.current.error).not.toBeNull())
    expect(result.current.error?.message).toBe('latest unavailable')
    expect(result.current.latest).toBeNull()
  })

  test('error from currentVersion is not cleared by latestVersion success', async () => {
    const api = new Client({ baseUrl: 'http://test' })
    vi.spyOn(api, 'currentVersion').mockRejectedValue(new Error('current unavailable'))
    vi.spyOn(api, 'latestVersion').mockResolvedValue('2.0.0')

    const { result } = renderHook(() => useAppVersion(), { wrapper: makeWrapper(api) })

    await waitFor(() => expect(result.current.latest).toBe('2.0.0'))
    expect(result.current.error?.message).toBe('current unavailable')
    expect(result.current.current).toBeNull()
  })
})
