import React, { createContext, useCallback, useContext, useEffect, useState } from 'react'

type Options = NotificationOptions & {
  onClose?: (this: Notification, ev: Event) => void
  onClick?: (this: Notification, ev: Event) => void
  onError?: (this: Notification, ev: Event) => void
  onShow?: (this: Notification, ev: Event) => void
  autoClose?: number // auto close the notification after the specified number of milliseconds
}

type BrowserNotificationsContext = {
  granted: boolean // is the permission granted to show notifications
  request: () => Promise<boolean> // request the permission to show notifications
  show: (title: string, options?: Options) => Promise<Notification | null>
}

const browserNotificationsContext = createContext<BrowserNotificationsContext>({
  granted: false,
  request: () => {
    throw new Error('The BrowserNotificationsProvider is not initialized')
  },
  show: () => {
    throw new Error('The BrowserNotificationsProvider is not initialized')
  },
})

/**
 * @link https://developer.mozilla.org/en-US/docs/Web/API/Notification
 */
export const BrowserNotificationsProvider = ({ children }: { children: React.JSX.Element }) => {
  const [granted, setGranted] = useState<boolean>(Notification?.permission === 'granted')

  // request the permission to show notifications from the user
  const request = useCallback(async (): Promise<boolean> => {
    // check if the permission is already granted
    if (!granted) {
      // ask the user for permission to show notifications
      const got: boolean = (await Notification?.requestPermission()) === 'granted'

      // update the state
      setGranted(got)

      return got
    }

    // since the permission is already granted, return true
    return true
  }, [granted])

  // show a notification
  const show = useCallback(
    async (title: string, options?: Options): Promise<Notification | null> => {
      // check if the permission is granted and request it if not
      if (!granted && !(await request())) {
        return null
      }

      const n = new Notification(title, options)

      if (options?.onClose) {
        n.onclose = options.onClose
      }

      if (options?.onClick) {
        n.onclick = options.onClick
      }

      if (options?.onError) {
        n.onerror = options.onError
      }

      if (options?.onShow) {
        n.onshow = options.onShow
      }

      if (options?.autoClose && options.autoClose > 0) {
        setTimeout(() => n.close(), options.autoClose)
      }

      return n
    },
    [granted, request]
  )

  // subscribe to the permission change event and update the granted state accordingly
  useEffect((): (() => void) => {
    const handler = (e: Event) => {
      if ((e.target && 'state' in e.target) || e.target instanceof PermissionStatus) {
        setGranted((e.target.state as PermissionState) === 'granted')
      }
    }

    let permissionStatus: PermissionStatus | null = null
    const eventName: keyof PermissionStatusEventMap = 'change'

    navigator?.permissions.query({ name: 'notifications' }).then((s) => {
      permissionStatus = s // store the status for use in the cleanup function

      s.addEventListener(eventName, handler)
    })

    // cleanup the event listener
    return () => permissionStatus?.removeEventListener(eventName, handler)
  }, [])

  return (
    <browserNotificationsContext.Provider value={{ granted, request, show }}>
      {children}
    </browserNotificationsContext.Provider>
  )
}

export const useBrowserNotifications = (): Readonly<BrowserNotificationsContext> => {
  const ctx = useContext(browserNotificationsContext)

  if (!ctx) {
    throw new Error('useBrowserNotifications must be used within a BrowserNotificationsProvider')
  }

  return ctx
}
