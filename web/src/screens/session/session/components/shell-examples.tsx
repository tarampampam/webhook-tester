import type { CodeHighlightTabsCode, CodeHighlightTabsProps } from '@mantine/code-highlight'
import { CodeHighlightTabs } from '@mantine/code-highlight'
import { IconBrandDebian, IconBrandWindows } from '@tabler/icons-react'
import React, { useMemo } from 'react'
import { L10nKey, useL10n, useStorage } from '~/shared'

const SHELL_UNIX_ICON = <IconBrandDebian size="1.2em" />

export const ShellExamples = ({
  url,
  ...props
}: Omit<CodeHighlightTabsProps, 'code'> & { url: Readonly<URL> }): React.JSX.Element => {
  const { t } = useL10n()
  const [activeTab, setActiveTab] = useStorage<number>(0, 'session-shell-examples-active-tab')

  const tabs = useMemo<Array<CodeHighlightTabsCode>>(() => {
    const asString = url.toString()

    return [
      {
        fileName: 'curl',
        language: 'bash',
        code: `curl -v --json '{"foo": "bar"}' '${asString}'`,
        icon: SHELL_UNIX_ICON,
      },
      {
        fileName: 'wget',
        language: 'bash',
        code: `wget -O- --post-data='{"foo": "bar"}' '${asString}'`,
        icon: SHELL_UNIX_ICON,
      },
      {
        fileName: 'HTTPie',
        language: 'bash',
        code: `http POST '${asString}' foo=bar --verbose`,
        icon: SHELL_UNIX_ICON,
      },
      {
        fileName: 'xh',
        language: 'bash',
        code: `xh POST '${asString}' foo=bar --verbose`,
        icon: SHELL_UNIX_ICON,
      },
      {
        fileName: 'PowerShell',
        language: 'bash',
        code: `irm '${asString}' -Method POST -Body '{"foo": "bar"}' -Verbose`,
        icon: <IconBrandWindows size="1.2em" />,
      },
    ]
  }, [url])

  return (
    <CodeHighlightTabs
      withCopyButton
      w="100%"
      copyLabel={t(L10nKey.copy)}
      copiedLabel={t(L10nKey.copied)}
      radius="sm"
      onTabChange={setActiveTab}
      activeTab={Math.min(activeTab ?? 0, tabs.length - 1)}
      {...props}
      code={tabs}
    />
  )
}
