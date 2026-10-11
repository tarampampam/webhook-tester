import React from 'react'

/**
 * Renders children wrapped in a wrapper component if the condition is true, otherwise renders children as they are.
 *
 * @example
 * ```tsx
 * <When condition={isLoggedIn} wrapper={(children) => <a href="/profile">{children}</a>}>
 *   My Profile
 * </When>
 * ```
 */
export const When = ({
  condition,
  wrapper,
  children,
}: {
  condition: boolean
  wrapper: (children: React.ReactNode) => React.JSX.Element
  children: React.ReactNode
}) => (condition ? wrapper(children) : children)
