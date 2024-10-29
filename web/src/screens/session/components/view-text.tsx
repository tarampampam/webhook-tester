import { CodeHighlight } from '@mantine/code-highlight'
import { Alert, Code } from '@mantine/core'
import { IconInfoCircle } from '@tabler/icons-react'
import React, { useEffect, useState } from 'react'

const decoder = new TextDecoder('utf-8')
const cutMessage = '\n\n[...content truncated (to view the full content, please download the binary file)...]\n\n'

export default function ViewText({
  input,
  contentType = null,
  lengthLimit = 1024 * 128, // 128KB
}: {
  input: Uint8Array | null
  contentType: string | null
  lengthLimit?: number
}): React.JSX.Element {
  const [content, setContent] = useState<string | null>(null)
  const [language, setLanguage] = useState<'json' | 'xml' | null>(null)
  const [trimmed, setTrimmed] = useState<boolean>(false)

  useEffect(() => {
    if (!input || input.length === 0) {
      setContent('// empty request body')
      setTrimmed(false)
      setLanguage('json')

      return
    }

    if (input.length > lengthLimit + cutMessage.length) {
      const [start, end] = [input.slice(0, lengthLimit / 2), input.slice(-lengthLimit / 2)]

      setContent(decoder.decode(start) + cutMessage + decoder.decode(end))
      setTrimmed(true)
      setLanguage(null)

      return
    }

    const [maybePretty, lang] = tryToFormat(decoder.decode(input), contentType)

    setTrimmed(false)
    setContent(maybePretty)
    setLanguage(lang)
  }, [input, lengthLimit, contentType])

  return (
    <>
      {trimmed && (
        <Alert color="yellow" my="sm" title="Data trimmed" icon={<IconInfoCircle />}>
          The request body is large and has been trimmed to {lengthLimit} bytes for performance reasons.
        </Alert>
      )}
      {!!content && language ? <CodeHighlight code={content} language={language} /> : <Code block>{content}</Code>}
    </>
  )
}

const tryToFormat = (
  str: string,
  contentType: string | null
): [string /* content, probably well-formatted */, 'json' | 'xml' | null /* language */] => {
  let looksLikeJson = false
  let looksLikeXml = false

  // try to determine format by content type
  if (contentType) {
    const clear = contentType.toLowerCase()

    looksLikeJson = clear.includes('json')
    looksLikeXml = clear.includes('xml')
  }

  // otherwise, try to determine format by content
  if (!looksLikeJson && !looksLikeXml) {
    const clear = str.trimStart()

    looksLikeJson = clear.length > 0 && (clear[0] === '{' || clear[0] === '[' || clear[0] === '"')
    looksLikeXml = clear.length > 0 && clear[0] === '<'
  }

  if (looksLikeJson) {
    try {
      const parsed = JSON.parse(str)

      return [JSON.stringify(parsed, undefined, 2), 'json']
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
    } catch (_) {
      // wrong json
    }
  } else if (looksLikeXml) {
    try {
      new DOMParser().parseFromString(str, 'text/xml')

      return [str, 'xml']
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
    } catch (_) {
      // wrong xml
    }
  }

  return [str, null] // return as is
}
