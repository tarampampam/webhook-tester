import type { HLJSApi, Language } from 'highlight.js'

// language modules
import plainText from 'highlight.js/lib/languages/plaintext'
import typescript from 'highlight.js/lib/languages/typescript'
import javascript from 'highlight.js/lib/languages/javascript'
import json from 'highlight.js/lib/languages/json'
import bash from 'highlight.js/lib/languages/bash'
import xml from 'highlight.js/lib/languages/xml'
import csharp from 'highlight.js/lib/languages/csharp'
import ruby from 'highlight.js/lib/languages/ruby'
import php from 'highlight.js/lib/languages/php'
import go from 'highlight.js/lib/languages/go'
import java from 'highlight.js/lib/languages/java'
import python from 'highlight.js/lib/languages/python'

const languages: Record<
  string,
  {
    aliases: string[]
    loader: () => (lib: HLJSApi) => Language
  }
> = {
  plaintext: { aliases: ['plain', 'text', 'txt'], loader: () => plainText },
  typescript: { aliases: [], loader: () => typescript },
  javascript: { aliases: ['js'], loader: () => javascript },
  json: { aliases: [], loader: () => json },
  bash: { aliases: [], loader: () => bash },
  xml: { aliases: [], loader: () => xml },
  csharp: { aliases: [], loader: () => csharp },
  ruby: { aliases: [], loader: () => ruby },
  php: { aliases: [], loader: () => php },
  go: { aliases: ['golang'], loader: () => go },
  java: { aliases: ['ebanina'], loader: () => java },
  python: { aliases: ['py'], loader: () => python },
  url: { aliases: ['url', 'uri'], loader: () => urlLanguage },
}

export const initializeHighlightJs = (lib: HLJSApi): void => {
  for (const [lang, config] of Object.entries(languages)) {
    const langFn = config.loader()

    void [lang, ...config.aliases].forEach((alias) => lib.registerLanguage(alias, langFn))
  }
}

const urlLanguage = (): Language => {
  const PCT = { scope: 'symbol', match: /%[0-9A-Fa-f]{2}/, relevance: 0 }

  return {
    name: 'URL',
    aliases: ['uri', 'url'],
    case_insensitive: true,
    contains: [
      { begin: [/[a-zA-Z][a-zA-Z0-9+\-.]*/, /:\/\//], beginScope: { 1: 'keyword', 2: 'comment' }, relevance: 10 },
      {
        begin: [/[^:@/?#\s]+/, /:/, /[^@/?#\s]+/, /@/],
        beginScope: { 1: 'attribute', 2: 'comment', 3: 'string', 4: 'comment' },
        relevance: 5,
      },
      {
        scope: 'built_in',
        match:
          /(?:(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}|[a-zA-Z][a-zA-Z0-9-]*|\d{1,3}(?:\.\d{1,3}){3})(?=[:/?#]|$)/,
        relevance: 0,
      },
      { begin: [/:/, /\d+/], beginScope: { 1: 'comment', 2: 'number' }, relevance: 0 },
      { begin: [/\//, /[^/?#;]*/], beginScope: { 1: 'deletion', 2: 'string' }, contains: [PCT], relevance: 0 },
      {
        begin: [/;/, /[^=;/?#]+/, /=/, /[^;/?#]*/],
        beginScope: { 1: 'comment', 2: 'attribute', 3: 'comment', 4: 'string' },
        relevance: 0,
      },
      { scope: 'comment', match: /[?&]/, relevance: 0 },
      {
        begin: [/[^=&#[\]\s]+(?:\[[^\]]*])*/, /=/, /[^&#]*/],
        beginScope: { 1: 'keyword', 2: 'comment', 3: 'attribute' },
        relevance: 0,
      },
      { begin: [/#/, /.*/], beginScope: { 1: 'comment', 2: 'code' }, relevance: 1 },
      PCT,
    ],
  }
}
