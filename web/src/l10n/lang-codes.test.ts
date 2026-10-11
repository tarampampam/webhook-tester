import { describe, expect, test } from 'vitest'
import { isLangCode } from './lang-codes'

describe('isLangCode', () => {
  test.each(['en', 'ru', 'zh', 'fr', 'de'])('accepts 2-letter lowercase code %s', (code) => {
    expect(isLangCode(code)).toBe(true)
  })

  test.each(['en-us', 'zh-hant', 'zh-hant-tw', 'sr-latn-rs', 'en-001'])(
    'accepts BCP 47 tag with subtag(s) %s',
    (code) => {
      expect(isLangCode(code)).toBe(true)
    }
  )

  test.each(['EN', 'en-US', 'e', 'eng', '', '12', 'en_us', 'en-', 'en-toolonggg'])(
    'rejects invalid code %s',
    (code) => {
      expect(isLangCode(code)).toBe(false)
    }
  )
})
