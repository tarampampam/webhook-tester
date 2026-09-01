import { colorsTuple, createTheme, type MantineColor, virtualColor } from '@mantine/core'

export const ThemeColor = {
  PureWhite: 'pure-white',
  NavbarBg: 'navbar-bg',
  NavbarText: 'navbar-text',
  LogoText: 'logo-text',
} as const satisfies Record<string, MantineColor>

export type ThemeColor = (typeof ThemeColor)[keyof typeof ThemeColor]

/**
 * The app theme. It extends the default Mantine theme.
 */
export const appTheme = createTheme({
  defaultRadius: 'sm',
  colors: {
    [ThemeColor.PureWhite]: colorsTuple('#ffffff'),
    [ThemeColor.NavbarBg]: virtualColor({
      name: ThemeColor.NavbarBg,
      light: 'cyan',
      dark: 'dark',
    }),
    [ThemeColor.NavbarText]: virtualColor({
      name: ThemeColor.NavbarText,
      light: ThemeColor.PureWhite,
      dark: 'light',
    }),
    [ThemeColor.LogoText]: virtualColor({
      name: ThemeColor.LogoText,
      light: ThemeColor.PureWhite,
      dark: ThemeColor.PureWhite,
    }),
  },
})
