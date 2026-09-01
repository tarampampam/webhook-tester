import { describe, expect, test } from 'vitest'
import { KEY, SUPPORTED_LANG_CODES, isSupportedLangCode, translate } from './l10n'

describe('SUPPORTED_LANG_CODES', () => {
  test('contains en and ru', () => {
    expect(SUPPORTED_LANG_CODES).toContain('en')
    expect(SUPPORTED_LANG_CODES).toContain('ru')
  })
})

describe('isSupportedLangCode', () => {
  test.each(['en', 'ru'])('returns true for supported code %s', (code) => {
    expect(isSupportedLangCode(code)).toBe(true)
  })

  test.each(['de', 'fr'])('returns false for valid but unsupported code %s', (code) => {
    expect(isSupportedLangCode(code)).toBe(false)
  })

  test.each(['EN', 'RU', ''])('returns false for invalid code %s', (code) => {
    expect(isSupportedLangCode(code)).toBe(false)
  })
})

describe('translate', () => {
  test('en and ru translations differ for every key', () => {
    for (const key of Object.values(KEY)) {
      if (key === KEY.localhost) {
        continue // skip this key as it is the same in both languages
      }

      expect(translate(key, 'en')).not.toBe(translate(key, 'ru'))
    }
  })

  test.each([
    [KEY.notFoundMessage, 'The requested page was not found'],
    [KEY.goToHome, 'Go to Home'],
  ] satisfies [KEY, string][])('en: %s → %s', (key, expected) => {
    expect(translate(key, 'en')).toBe(expected)
  })

  test.each([
    [KEY.notFoundMessage, 'Страница не найдена'],
    [KEY.goToHome, 'На главную'],
  ] satisfies [KEY, string][])('ru: %s → %s', (key, expected) => {
    expect(translate(key, 'ru')).toBe(expected)
  })
})
