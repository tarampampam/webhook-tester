import React from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { BrowserNotificationsProvider, useBrowserNotifications } from './browser-notifications'

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <BrowserNotificationsProvider>
    <>{children}</>
  </BrowserNotificationsProvider>
)

describe('BrowserNotificationsProvider', () => {
  let queryMock: ReturnType<typeof vi.fn>
  let capturedChangeHandler: EventListener | null
  let permissionStatus: {
    state: PermissionState
    addEventListener: ReturnType<typeof vi.fn>
    removeEventListener: ReturnType<typeof vi.fn>
  }
  let notificationInstance: { close: ReturnType<typeof vi.fn> }
  let requestPermission: ReturnType<typeof vi.fn>
  let NotificationMock: ReturnType<typeof vi.fn> & {
    permission: NotificationPermission
    requestPermission: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    capturedChangeHandler = null
    permissionStatus = {
      state: 'prompt',
      addEventListener: vi.fn((_, h: EventListener) => {
        capturedChangeHandler = h
      }),
      removeEventListener: vi.fn(),
    }
    queryMock = vi.fn().mockResolvedValue(permissionStatus)
    vi.spyOn(navigator, 'permissions', 'get').mockReturnValue({ query: queryMock } as unknown as Permissions)

    notificationInstance = { close: vi.fn() }
    requestPermission = vi.fn().mockResolvedValue('default')
    const initialPermission: NotificationPermission = 'default'
    NotificationMock = Object.assign(
      vi.fn(function () {
        return notificationInstance
      }),
      {
        permission: initialPermission,
        requestPermission,
      }
    )
    vi.stubGlobal('Notification', NotificationMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  test('request() returns true without prompting when already granted', async () => {
    NotificationMock.permission = 'granted'
    const { result } = renderHook(() => useBrowserNotifications(), { wrapper })
    await act(async () => {})

    expect(await result.current.request()).toBe(true)
    expect(requestPermission).not.toHaveBeenCalled()
  })

  test('request() prompts the user, updates granted, and returns true on grant', async () => {
    requestPermission.mockResolvedValue('granted')
    const { result } = renderHook(() => useBrowserNotifications(), { wrapper })
    await act(async () => {})

    await act(async () => {
      await result.current.request()
    })

    expect(result.current.granted).toBe(true)
  })

  test('show() creates and returns a Notification when permission is granted', async () => {
    NotificationMock.permission = 'granted'
    const { result } = renderHook(() => useBrowserNotifications(), { wrapper })
    await act(async () => {})

    let n: Notification | null | undefined
    await act(async () => {
      n = await result.current.show('Test', { body: 'hello' })
    })

    expect(NotificationMock).toHaveBeenCalledWith('Test', { body: 'hello' })
    expect(n).toBe(notificationInstance)
  })

  test('show() returns null without creating a Notification when permission is denied', async () => {
    requestPermission.mockResolvedValue('denied')
    const { result } = renderHook(() => useBrowserNotifications(), { wrapper })
    await act(async () => {})

    let n: Notification | null | undefined
    await act(async () => {
      n = await result.current.show('Test')
    })

    expect(n).toBeNull()
    expect(NotificationMock).not.toHaveBeenCalled()
  })

  test('show() closes the notification after autoClose ms', async () => {
    NotificationMock.permission = 'granted'
    const { result } = renderHook(() => useBrowserNotifications(), { wrapper })
    await act(async () => {})

    vi.useFakeTimers()
    await act(async () => {
      await result.current.show('Test', { autoClose: 500 })
    })
    vi.advanceTimersByTime(500)
    vi.useRealTimers()

    expect(notificationInstance.close).toHaveBeenCalledOnce()
  })

  test('granted updates reactively when the Permissions API fires a change event', async () => {
    const { result } = renderHook(() => useBrowserNotifications(), { wrapper })
    await act(async () => {})

    await act(async () => {
      capturedChangeHandler?.({ target: { state: 'granted' } } as unknown as Event)
    })

    expect(result.current.granted).toBe(true)
  })

  test('does not register the change listener when unmount happens before query resolves', async () => {
    let resolveQuery: (s: PermissionStatus) => void = () => {}
    queryMock.mockReturnValueOnce(
      new Promise<PermissionStatus>((r) => {
        resolveQuery = r
      })
    )

    const { unmount } = renderHook(() => useBrowserNotifications(), { wrapper })
    unmount()

    await act(async () => {
      resolveQuery(permissionStatus as unknown as PermissionStatus)
    })

    expect(permissionStatus.addEventListener).not.toHaveBeenCalled()
  })
})
