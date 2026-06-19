import { Box, type BoxProps, Indicator, Text } from '@mantine/core'
import React from 'react'
import { L10nKey, useAppVersion, useL10n, When } from '~/shared'

/**
 * A component that displays the current version of the application and indicates if an update is available.
 */
export const VersionBox = ({ ...props }: BoxProps): React.JSX.Element => {
  const { current, latest, updateAvailable } = useAppVersion()
  const { t } = useL10n()

  return (
    <Box {...props}>
      <When
        condition={!!updateAvailable}
        wrapper={(children) => (
          <Indicator size={6} color="lime" processing>
            {children}
          </Indicator>
        )}
      >
        <Text fz="xs">
          <When
            condition={!!updateAvailable && !!latest}
            wrapper={(children) => (
              <a
                href={__LATEST_RELEASE_LINK__}
                title={t(L10nKey.appUpdateAvailable) + ': v' + latest}
                target="_blank"
                rel="noreferrer"
              >
                {children}
              </a>
            )}
          >
            v{current ?? '…'}
          </When>
        </Text>
      </When>
    </Box>
  )
}
