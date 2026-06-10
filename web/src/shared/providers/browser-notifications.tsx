import React, { createContext, useCallback, useContext, useEffect, useState } from 'react'

type Options = NotificationOptions & {
  onClose?: (this: Notification, ev: Event) => void
  onClick?: (this: Notification, ev: Event) => void
  onError?: (this: Notification, ev: Event) => void
  onShow?: (this: Notification, ev: Event) => void
  autoClose?: number // auto close the notification after the specified number of milliseconds
}

type BrowserNotificationsContext = {
  /** Whether permission to show notifications has been granted. */
  readonly granted: boolean

  /**
   * Request permission to show notifications. Returns true if permission is granted, false otherwise.
   * If permission is already granted, it resolves to true immediately without prompting the user.
   */
  request: () => Promise<boolean>

  /**
   * Show a browser notification with the given title and options. If permission is not granted, it will first
   * request permission.
   * Returns the Notification object if shown successfully, or null if permission was denied or an error occurred.
   */
  show: (title: string, options?: Options) => Promise<Notification | null>
}

const ctx = createContext<BrowserNotificationsContext | null>(null)

/**
 * Provides browser notification state and helpers to the component tree. Tracks
 * permission status reactively via the Permissions API and exposes `request` and
 * `show` methods. Wrap your app (or the relevant subtree) with this before calling
 * `useBrowserNotifications`.
 *
 * @link https://developer.mozilla.org/en-US/docs/Web/API/Notification
 */
export const BrowserNotificationsProvider = ({ children }: { children: React.JSX.Element }) => {
  const [granted, setGranted] = useState<boolean>(Notification?.permission === 'granted')

  const request = useCallback(async (): Promise<boolean> => {
    if (granted) {
      return true
    }

    const got = (await Notification?.requestPermission()) === 'granted'
    setGranted(got)

    return got
  }, [granted])

  const show = useCallback(
    async (title: string, options?: Options): Promise<Notification | null> => {
      if (!granted && !(await request())) {
        return null
      }

      const n = new Notification(title, options)

      n.onclose = options?.onClose || null
      n.onclick = options?.onClick || null
      n.onerror = options?.onError || null
      n.onshow = options?.onShow || null

      if (options?.autoClose && options.autoClose > 0) {
        const t = setTimeout(() => {
          n.close()
          clearTimeout(t)
        }, options.autoClose)
      }

      return n
    },
    [granted, request]
  )

  useEffect(() => {
    const handler = (e: Event) => {
      if ((e.target && 'state' in e.target) || e.target instanceof PermissionStatus) {
        setGranted((e.target.state as PermissionState) === 'granted')
      }
    }

    let permissionStatus: PermissionStatus | null = null
    const eventName: keyof PermissionStatusEventMap = 'change'
    let cancelled = false

    navigator?.permissions
      .query({ name: 'notifications' })
      .then((s) => {
        if (cancelled) {
          return
        }

        permissionStatus = s // store the status for use in the cleanup function

        s.addEventListener(eventName, handler)
      })
      .catch(() => {
        /* notifications permission name not supported in this browser */
      })

    // cleanup the event listener
    return () => {
      cancelled = true
      permissionStatus?.removeEventListener(eventName, handler)
    }
  }, [])

  return <ctx.Provider value={{ granted, request, show }}>{children}</ctx.Provider>
}

/** Returns browser notification state and helpers. Must be used inside BrowserNotificationsProvider. */
export const useBrowserNotifications = (): Readonly<BrowserNotificationsContext> => {
  const context = useContext(ctx)
  if (!context) {
    throw new Error('useBrowserNotifications must be used within a BrowserNotificationsProvider')
  }

  return context
}
