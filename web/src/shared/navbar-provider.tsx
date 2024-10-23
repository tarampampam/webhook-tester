import React, { createContext, useContext } from 'react'

export type NavBarContext = {
  readonly component: React.ReactNode
  setComponent: (component: React.ReactNode) => void
}

/** The NavBar context. */
const navBarContext = createContext<NavBarContext>({
  component: null,
  setComponent: () => {
    throw new Error('NavBarProvider is not initialized')
  },
})

/** The provider for the NavBar context. */
export default function NavBarProvider({ children }: { children: React.ReactNode }): React.ReactNode {
  const [component, setComponent] = React.useState<React.ReactNode>(null)

  return <navBarContext.Provider value={{ component, setComponent }}>{children}</navBarContext.Provider>
}

/** A hook to access the NavBar context. */
export function useNavBar(): NavBarContext {
  const ctx = useContext(navBarContext)

  if (!ctx) {
    throw new Error('useNavBar must be used within a NavBarProvider')
  }

  return ctx
}
