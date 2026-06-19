import { describe, expect, test } from 'vitest'
import { ThemeColor, appTheme } from './theme'

describe('appTheme', () => {
  test('defaultRadius is sm', () => {
    expect(appTheme.defaultRadius).toBe('sm')
  })

  test('all ThemeColor keys are defined in colors', () => {
    const keys = Object.keys(appTheme.colors ?? {})
    expect(keys).toContain(ThemeColor.PureWhite)
    expect(keys).toContain(ThemeColor.NavbarBg)
    expect(keys).toContain(ThemeColor.NavbarText)
    expect(keys).toContain(ThemeColor.LogoText)
  })

  test('pure-white has 10 shades all set to #ffffff', () => {
    const shades = appTheme.colors?.[ThemeColor.PureWhite]
    expect(shades).toBeDefined()
    expect(shades).toHaveLength(10)
    for (const shade of shades ?? []) {
      expect(shade).toBe('#ffffff')
    }
  })
})
