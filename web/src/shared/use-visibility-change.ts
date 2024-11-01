import { useEffect, useState } from 'react'

/**
 * Hook that returns the visibility state of the document.
 *
 * @link https://developer.mozilla.org/en-US/docs/Web/API/Document/visibilitychange_event
 */
export function useVisibilityChange(): boolean {
  const [isVisible, setIsVisible] = useState<boolean>(document.visibilityState === 'visible')

  useEffect(() => {
    const handleVisibilityChange = () => setIsVisible(document.visibilityState === 'visible')

    document.addEventListener('visibilitychange', handleVisibilityChange)

    return () => document.removeEventListener('visibilitychange', handleVisibilityChange)
  }, [])

  return isVisible
}
