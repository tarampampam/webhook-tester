import React, { createContext, type PropsWithChildren, useCallback, useContext, useEffect, useMemo } from 'react'
import { isSupportedLangCode, KEY, SUPPORTED_LANG_CODES, type SupportedLangCode, translate } from '~/l10n'
import { useStorage } from '~/shared/hooks/use-storage'

/** Re-exports the KEY translation key constant/type - use it instead of importing KEY directly from ~/l10n. */
export const L10nKey = KEY

/** Shape of the l10n context value exposed to consumers via useL10n(). */
interface Context {
  /** Currently active locale code. */
  readonly langCode: SupportedLangCode

  /** All supported locale codes. */
  readonly supported: ReadonlyArray<SupportedLangCode>

  /** Maps supported locale codes to their display names in English. */
  readonly langNames: ReadonlyMap<SupportedLangCode, string>

  /** Switches the active locale. */
  switchTo(lang: SupportedLangCode): void

  /** Returns the translated string for the given message key in the active locale. */
  t(key: KEY): string

  /** Returns a human-friendly relative time string (e.g. "3 minutes ago") for the given dates in the active locale. */
  relativeTime(a: Date, b: Date): string

  /**
   * Returns a localized string representation of the given date in the active locale, formatted according to
   * the given options.
   */
  formatDateTime(date: Date, opts: Intl.DateTimeFormatOptions): string
}

const ctx = createContext<Context | null>(null)

const RELATIVE_TIME_UNITS: Array<[Intl.RelativeTimeFormatUnit, number]> = [
  ['year', 365 * 24 * 60 * 60 * 1000],
  ['month', 30 * 24 * 60 * 60 * 1000],
  ['week', 7 * 24 * 60 * 60 * 1000],
  ['day', 24 * 60 * 60 * 1000],
  ['hour', 60 * 60 * 1000],
  ['minute', 60 * 1000],
  ['second', 1000],
]

/**
 * Provides the active locale and the t() translation helper to the component tree.
 *
 * Detects locale from browser preferences on mount; falls back to 'en' if none match a supported locale.
 */
export const L10nProvider = ({ children }: PropsWithChildren): React.JSX.Element => {
  const [storedLang, setStoredLang] = useStorage<string | null>(null, 'l10n-locale', 'local')

  const browserLang = useMemo<SupportedLangCode>(
    () =>
      (typeof navigator !== 'undefined'
        ? navigator.languages && navigator.languages.length
          ? navigator.languages
          : [navigator.language]
        : []
      )
        .flatMap((l) => {
          const lower = l.toLowerCase()
          const base = lower.split('-')[0] ?? lower
          return base !== lower ? [lower, base] : [lower]
        })
        .find(isSupportedLangCode) ?? 'en',
    []
  )

  const langCode = useMemo<SupportedLangCode>(() => {
    if (storedLang !== null && isSupportedLangCode(storedLang)) {
      return storedLang
    }

    return browserLang
  }, [storedLang, browserLang])

  useEffect(() => {
    if (storedLang !== null && !isSupportedLangCode(storedLang)) {
      setStoredLang(null)
    }
  }, [storedLang, setStoredLang])

  useEffect(() => {
    if (typeof document !== 'undefined' && 'documentElement' in document) {
      document.documentElement.setAttribute('lang', langCode)
    }
  }, [langCode])

  const switchTo = useCallback((lang: SupportedLangCode) => setStoredLang(lang), [setStoredLang])

  const langNames = useMemo<ReadonlyMap<SupportedLangCode, string>>(() => {
    const resolver = (lang: SupportedLangCode): string => {
      try {
        return new Intl.DisplayNames(['en'], { type: 'language' }).of(lang) ?? lang
      } catch {
        return lang
      }
    }

    return new Map(SUPPORTED_LANG_CODES.map((code) => [code, resolver(code)]))
  }, [])

  const t = useCallback((key: KEY): string => translate(key, langCode), [langCode])

  const relativeTime = useCallback(
    (a: Date, b: Date): string => {
      try {
        const rtf = new Intl.RelativeTimeFormat(langCode, { numeric: 'auto' })
        const diff = a.getTime() - b.getTime()

        for (const [unit, ms] of RELATIVE_TIME_UNITS) {
          if (Math.abs(diff) >= ms) {
            return rtf.format(Math.round(diff / ms), unit)
          }
        }

        return translate(L10nKey.justNow, langCode) || rtf.format(0, 'second')
      } catch {
        return a.toLocaleDateString(langCode) // fall back to absolute date
      }
    },
    [langCode]
  )

  const formatDateTime = useCallback(
    (date: Date, opts: Intl.DateTimeFormatOptions): string => {
      try {
        return new Intl.DateTimeFormat(langCode, opts).format(date)
      } catch {
        return date.toLocaleString(langCode) // fall back to default formatting
      }
    },
    [langCode]
  )

  return (
    <ctx.Provider
      value={{
        langCode,
        supported: SUPPORTED_LANG_CODES,
        langNames,
        switchTo,
        t,
        relativeTime,
        formatDateTime,
      }}
    >
      {children}
    </ctx.Provider>
  )
}

/** Returns the l10n context. Must be called within an L10nProvider subtree. */
export const useL10n = (): Context => {
  const context = useContext(ctx)
  if (!context) {
    throw new Error('useL10n must be used within a L10nProvider')
  }

  return context
}
