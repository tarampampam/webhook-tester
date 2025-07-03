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
}

export const initializeHighlightJs = (lib: HLJSApi): void => {
  for (const [lang, config] of Object.entries(languages)) {
    const langFn = config.loader()

    ;[lang, ...config.aliases].forEach((alias) => lib.registerLanguage(alias, langFn))
  }
}
