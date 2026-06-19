import { act, renderHook } from '@testing-library/react'
import React from 'react'
import { afterEach, describe, expect, test, vi } from 'vitest'
import { KEY, SUPPORTED_LANG_CODES } from '~/l10n'
import { L10nKey, L10nProvider, useL10n } from './l10n'

const wrapper = ({ children }: { children: React.ReactNode }) => <L10nProvider>{children}</L10nProvider>

describe('L10nKey', () => {
  test('is the KEY constant re-exported from ~/l10n', () => {
    expect(L10nKey).toBe(KEY)
  })
})

describe('useL10n', () => {
  test('throws when called outside an L10nProvider', () => {
    expect(() => renderHook(() => useL10n())).toThrow('useL10n must be used within a L10nProvider')
  })
})

describe('L10nProvider', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  describe('langCode', () => {
    test('detects supported locale from navigator.languages', () => {
      vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['ru'])
      const { result } = renderHook(() => useL10n(), { wrapper })
      expect(result.current.langCode).toBe('ru')
    })

    test('strips region subtag and matches the base lang code', () => {
      vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['ru-RU'])
      const { result } = renderHook(() => useL10n(), { wrapper })
      expect(result.current.langCode).toBe('ru')
    })

    test('picks the first matching supported locale in the preference list', () => {
      vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['fr', 'de', 'ru'])
      const { result } = renderHook(() => useL10n(), { wrapper })
      expect(result.current.langCode).toBe('ru')
    })

    test('falls back to navigator.language when navigator.languages is empty', () => {
      vi.spyOn(navigator, 'languages', 'get').mockReturnValue([])
      vi.spyOn(navigator, 'language', 'get').mockReturnValue('ru')
      const { result } = renderHook(() => useL10n(), { wrapper })
      expect(result.current.langCode).toBe('ru')
    })

    test('falls back to en when no navigator locale matches a supported code', () => {
      vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['fr', 'de'])
      const { result } = renderHook(() => useL10n(), { wrapper })
      expect(result.current.langCode).toBe('en')
    })
  })

  describe('t', () => {
    test.each([
      { langCode: 'en', key: KEY.goToHome, want: 'Go to Home' },
      { langCode: 'en', key: KEY.selectLanguage, want: 'Select language' },
      { langCode: 'ru', key: KEY.goToHome, want: 'На главную' },
      { langCode: 'ru', key: KEY.selectLanguage, want: 'Выберите язык' },
    ])('$langCode/$key → "$want"', ({ langCode, key, want }) => {
      vi.spyOn(navigator, 'languages', 'get').mockReturnValue([langCode])
      const { result } = renderHook(() => useL10n(), { wrapper })
      expect(result.current.t(key)).toBe(want)
    })
  })

  describe('switchTo', () => {
    test('updates langCode and rewires t() to the new locale', async () => {
      vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['en'])
      const { result } = renderHook(() => useL10n(), { wrapper })

      expect(result.current.langCode).toBe('en')
      expect(result.current.t(KEY.goToHome)).toBe('Go to Home')

      await act(async () => {
        result.current.switchTo('ru')
      })

      expect(result.current.langCode).toBe('ru')
      expect(result.current.t(KEY.goToHome)).toBe('На главную')
    })

    test('persists the selection and restores it after remount', async () => {
      vi.spyOn(navigator, 'languages', 'get').mockReturnValue(['en'])
      const { result, unmount } = renderHook(() => useL10n(), { wrapper })

      await act(async () => {
        result.current.switchTo('ru')
      })
      unmount()

      const { result: result2 } = renderHook(() => useL10n(), { wrapper })
      expect(result2.current.langCode).toBe('ru')
    })
  })

  describe('supported', () => {
    test('exposes all supported lang codes via supported', () => {
      const { result } = renderHook(() => useL10n(), { wrapper })
      expect(result.current.supported).toEqual(SUPPORTED_LANG_CODES)
    })
  })

  describe('langNames', () => {
    test('exposes a non-empty display name for every supported code', () => {
      const { result } = renderHook(() => useL10n(), { wrapper })
      SUPPORTED_LANG_CODES.forEach((code) => {
        const name = result.current.langNames.get(code)
        expect(name).toBeTypeOf('string')
        expect((name ?? '').length).toBeGreaterThan(0)
      })
    })

    test('falls back to the lang code when Intl.DisplayNames throws', () => {
      vi.spyOn(Intl.DisplayNames.prototype, 'of').mockImplementation(() => {
        throw new Error('mocked error')
      })
      const { result } = renderHook(() => useL10n(), { wrapper })
      SUPPORTED_LANG_CODES.forEach((code) => {
        expect(result.current.langNames.get(code)).toBe(code)
      })
    })

    test('falls back to the lang code when Intl.DisplayNames returns undefined', () => {
      vi.spyOn(Intl.DisplayNames.prototype, 'of').mockReturnValue(undefined)
      const { result } = renderHook(() => useL10n(), { wrapper })
      SUPPORTED_LANG_CODES.forEach((code) => {
        expect(result.current.langNames.get(code)).toBe(code)
      })
    })
  })

  describe('relativeTime', () => {
    const BASE = new Date('2024-01-15T12:00:00.000Z')
    const past = (ms: number): Date => new Date(BASE.getTime() - ms)
    const future = (ms: number): Date => new Date(BASE.getTime() + ms)

    test.each([
      { langCode: 'en', giveA: BASE, giveB: BASE, want: 'Just now' },
      { langCode: 'en', giveA: past(2_000), giveB: BASE, want: '2 seconds ago' },
      { langCode: 'en', giveA: past(59_999), giveB: BASE, want: '60 seconds ago' },
      { langCode: 'en', giveA: past(60_000), giveB: BASE, want: '1 minute ago' },
      { langCode: 'en', giveA: past(2 * 60_000), giveB: BASE, want: '2 minutes ago' },
      { langCode: 'en', giveA: past(3_599_999), giveB: BASE, want: '60 minutes ago' },
      { langCode: 'en', giveA: past(3_600_000), giveB: BASE, want: '1 hour ago' },
      { langCode: 'en', giveA: past(2 * 3_600_000), giveB: BASE, want: '2 hours ago' },
      { langCode: 'en', giveA: past(86_400_000), giveB: BASE, want: 'yesterday' },
      { langCode: 'en', giveA: past(2 * 86_400_000), giveB: BASE, want: '2 days ago' },
      { langCode: 'en', giveA: past(7 * 86_400_000), giveB: BASE, want: 'last week' },
      { langCode: 'en', giveA: past(2 * 7 * 86_400_000), giveB: BASE, want: '2 weeks ago' },
      { langCode: 'en', giveA: past(30 * 86_400_000), giveB: BASE, want: 'last month' },
      { langCode: 'en', giveA: past(2 * 30 * 86_400_000), giveB: BASE, want: '2 months ago' },
      { langCode: 'en', giveA: past(365 * 86_400_000), giveB: BASE, want: 'last year' },
      { langCode: 'en', giveA: past(2 * 365 * 86_400_000), giveB: BASE, want: '2 years ago' },
      { langCode: 'en', giveA: future(2 * 365 * 86_400_000), giveB: BASE, want: 'in 2 years' },
      { langCode: 'ru', giveA: BASE, giveB: BASE, want: 'Только что' },
      { langCode: 'ru', giveA: past(2_000), giveB: BASE, want: '2 секунды назад' },
      { langCode: 'ru', giveA: past(60_000), giveB: BASE, want: '1 минуту назад' },
      { langCode: 'ru', giveA: past(2 * 60_000), giveB: BASE, want: '2 минуты назад' },
      { langCode: 'ru', giveA: past(3_600_000), giveB: BASE, want: '1 час назад' },
      { langCode: 'ru', giveA: past(2 * 3_600_000), giveB: BASE, want: '2 часа назад' },
      { langCode: 'ru', giveA: past(86_400_000), giveB: BASE, want: 'вчера' },
      { langCode: 'ru', giveA: past(2 * 86_400_000), giveB: BASE, want: 'позавчера' },
      { langCode: 'ru', giveA: past(7 * 86_400_000), giveB: BASE, want: 'на прошлой неделе' },
      { langCode: 'ru', giveA: past(30 * 86_400_000), giveB: BASE, want: 'в прошлом месяце' },
      { langCode: 'ru', giveA: past(365 * 86_400_000), giveB: BASE, want: 'в прошлом году' },
    ])('relativeTime $langCode: $want', ({ langCode, giveA, giveB, want }) => {
      vi.spyOn(navigator, 'languages', 'get').mockReturnValue([langCode])
      const { result } = renderHook(() => useL10n(), { wrapper })
      expect(result.current.relativeTime(giveA, giveB)).toBe(want)
    })
  })
})
