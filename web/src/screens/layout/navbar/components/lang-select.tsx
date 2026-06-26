import { type ComboboxStringGroupData, NativeSelect, type NativeSelectProps } from '@mantine/core'
import { IconLanguage } from '@tabler/icons-react'
import React, { useCallback, useMemo } from 'react'
import { isSupportedLangCode } from '~/l10n'
import { L10nKey, useL10n } from '~/shared'

export const LangSelect = ({ ...props }: NativeSelectProps): React.JSX.Element => {
  const { t, supported, switchTo, langCode, langNames } = useL10n()

  const data = useMemo<ComboboxStringGroupData>(() => {
    return supported.map((code) => ({
      value: code,
      label: langNames.get(code) ?? code,
    }))
  }, [supported, langNames])

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      const lang = e.target.value
      if (isSupportedLangCode(lang)) {
        switchTo(lang)
      }
    },
    [switchTo]
  )

  return (
    <NativeSelect
      leftSection={<IconLanguage size="1.1em" />}
      description={t(L10nKey.selectLanguage)}
      data={data}
      onChange={handleChange}
      value={langCode}
      styles={{ description: { color: 'inherit' } }}
      {...props}
    ></NativeSelect>
  )
}
