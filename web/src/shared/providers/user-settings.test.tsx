import React from 'react'
import { beforeEach, describe, expect, test } from 'vitest'
import { act, render } from '@testing-library/react'
import { UserSettingsProvider, useUserSettings } from './user-settings'

const Consumer = (): React.JSX.Element => {
  const { userSettings, updateUserSettings } = useUserSettings()
  return (
    <>
      <div data-testid="settings">{JSON.stringify(userSettings)}</div>
      <button onClick={() => updateUserSettings({ showRequestDetails: false })}>update</button>
    </>
  )
}

describe('UserSettingsProvider', () => {
  beforeEach(() => localStorage.clear())

  test('exposes default settings to children', () => {
    const { getByTestId } = render(
      <UserSettingsProvider>
        <Consumer />
      </UserSettingsProvider>
    )

    expect(getByTestId('settings').textContent).toContain('"showRequestDetails":true')
    expect(getByTestId('settings').textContent).toContain('"autoNavigateToNewRequest":true')
    expect(getByTestId('settings').textContent).toContain('"showNativeRequestNotifications":false')
  })

  test('updateUserSettings merges a partial update without affecting other settings', async () => {
    const { getByTestId, getByRole } = render(
      <UserSettingsProvider>
        <Consumer />
      </UserSettingsProvider>
    )

    await act(async () => {
      getByRole('button').click()
    })

    expect(getByTestId('settings').textContent).toContain('"showRequestDetails":false')
    expect(getByTestId('settings').textContent).toContain('"autoNavigateToNewRequest":true')
  })
})
