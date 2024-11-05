import { render as testingLibraryRender } from '@testing-library/react'
import { MantineProvider } from '@mantine/core'

/** @link https://mantine.dev/guides/vitest/#custom-render */
export function render(ui: React.ReactNode) {
  return testingLibraryRender(<>{ui}</>, {
    wrapper: ({ children }: { children: React.ReactNode }) => <MantineProvider>{children}</MantineProvider>,
  })
}
