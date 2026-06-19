import type { CodeHighlightTabsCode, CodeHighlightTabsProps } from '@mantine/code-highlight'
import { CodeHighlightControl, CodeHighlightTabs } from '@mantine/code-highlight'
import { notifications } from '@mantine/notifications'
import { IconExternalLink, IconLink, IconRun } from '@tabler/icons-react'
import React, { useCallback, useMemo, useState } from 'react'
import { anyToError, L10nKey, useL10n } from '~/shared'

const TAB_ICON = <IconLink size="1em" />

export const WebhookURL = ({
  url,
  ...props
}: Omit<CodeHighlightTabsProps, 'code'> & { url: Readonly<URL> }): React.JSX.Element => {
  const { t } = useL10n()
  const [activeTab, setActiveTab] = useState<number>(0)

  const tabs = useMemo<Array<CodeHighlightTabsCode>>(
    () => [
      {
        fileName: t(L10nKey.webhookUrl),
        code: url.toString(),
        language: 'url',
        icon: TAB_ICON,
      },
      {
        fileName: t(L10nKey.withSpecificStatusCode),
        code: `${url.toString().replace(/\/$/, '')}/404`,
        language: 'url',
        icon: TAB_ICON,
      },
    ],
    [t, url]
  )

  const activeUrl = useMemo<URL>(() => {
    try {
      return new URL(tabs[activeTab]?.code ?? url.toString())
    } catch {
      return url
    }
  }, [activeTab, tabs, url])

  const handleSendRequest = useCallback(async () => {
    const u = new URL(activeUrl) // create a copy to modify

    try {
      const payload = {
        xhr: 'test',
        now: Math.floor(Date.now() / 1000),
      }

      const methods: readonly [string, ...string[]] = ['get', 'head', 'post', 'put', 'delete', 'patch']
      const method = methods[Math.floor(Math.random() * methods.length)] ?? methods[0]

      if (Math.random() > 0.5) {
        u.pathname = u.pathname.replace(/\/$/, '') + '/any/path' // add random path, 50% chance
      }

      u.searchParams.set('test', 'true')
      u.searchParams.set('now', String(payload.now))

      await fetch(
        new Request(u, {
          method: method.toUpperCase(), // pick random method
          cache: 'no-cache',
          headers: { 'Content-Type': 'application/json', 'X-Test-Header': 'test' },
          body: !['get', 'head'].includes(method) ? JSON.stringify(payload) : undefined,
        })
      )
    } catch (err) {
      notifications.show({
        title: t(L10nKey.somethingWentWrong),
        message: anyToError(err).message,
        color: 'orange',
      })
    }
  }, [activeUrl, t])

  const controls = useMemo<Array<React.ReactNode>>(
    () => [
      <CodeHighlightControl
        key="open_in_new_tab"
        component="a"
        href={activeUrl.toString()}
        target="_blank"
        rel="noopener noreferrer"
        tooltipLabel={t(L10nKey.openInNewTab)}
      >
        <IconExternalLink size="1em" />
      </CodeHighlightControl>,
      <CodeHighlightControl key="fetch" tooltipLabel={t(L10nKey.sendTestRequest)} onClick={handleSendRequest}>
        <IconRun size="1em" />
      </CodeHighlightControl>,
    ],
    [activeUrl, handleSendRequest, t]
  )

  return (
    <CodeHighlightTabs
      w="100%"
      withCopyButton
      copyLabel={t(L10nKey.copy)}
      copiedLabel={t(L10nKey.copied)}
      radius="sm"
      styles={{
        file: { fontWeight: 'lighter' },
        filesScrollarea: { right: `${28 * (controls.length + 1) + 8}px` },
      }}
      activeTab={activeTab}
      onTabChange={setActiveTab}
      controls={controls}
      {...props}
      code={tabs}
    />
  )
}
