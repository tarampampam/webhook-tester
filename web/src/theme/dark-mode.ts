/**
 * Checks if the user prefers dark mode, and allows to subscribe to changes.
 *
 * Not to forget to unsubscribe from the listener when the component is unmounted.
 *
 * @return A tuple with the current dark mode state and a function to unsubscribe from the listener.
 */
export const isDarkMode = (onChange?: (dark: boolean) => void): [boolean, () => void] => {
  const preferDarkSelector: string = '(prefers-color-scheme: dark)'

  const update = (event: MediaQueryListEvent): void => {
    if (onChange) {
      onChange(event.matches)
    }
  }

  let closer = () => {}

  if (window.matchMedia) {
    if (onChange) {
      window.matchMedia(preferDarkSelector).addEventListener('change', update, { passive: true })
      closer = () => window.matchMedia(preferDarkSelector).removeEventListener('change', update)
    }

    return [window.matchMedia(preferDarkSelector).matches, closer]
  }

  return [false, closer]
}
