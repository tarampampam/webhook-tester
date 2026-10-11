import type { BoxProps } from '@mantine/core'
import { Box, Group, Loader, ScrollArea, Stack, Text, useComputedColorScheme } from '@mantine/core'
import React, { useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { pathTo, ROUTE_ID, useActiveRequestID, useActiveSessionID } from '~/routing'
import { L10nKey, LogoText, useL10n, useRequests } from '~/shared'
import { ThemeColor } from '~/theme'
import { AnimatedStack } from './components/animated-stack'
import { DeleteAllButton } from './components/delete-all-button'
import { LangSelect } from './components/lang-select'
import { Request } from './components/request'
import { RequestsNavigator } from './components/requests-navigator'
import { VersionBox } from './components/version-box'

export const Navbar = (): React.JSX.Element => {
  const { requests, isReady } = useRequests()
  const [sID, rID] = [useActiveSessionID(), useActiveRequestID()]
  const navigate = useNavigate()

  const handleOutsideClick = useCallback(
    (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
      if ((e.target as HTMLElement).closest('[data-request-item]')) {
        return
      }

      if (sID) {
        navigate(pathTo(ROUTE_ID.Session, { sID }))
      }
    },
    [navigate, sID]
  )

  return (
    <Stack h="100%" gap="xs">
      <Stack w="100%" gap="xs">
        <Group c={ThemeColor.NavbarText} justify="space-between" gap="xs" wrap="nowrap" my="sm">
          <LogoText maw={170} />
          <VersionBox mt={5} />
        </Group>
      </Stack>

      {(!isReady && <RequestsNavigator.Skeleton opacity={0.5} />) || <RequestsNavigator />}

      {(!isReady && (
        <Stack w="100%" gap="xs" flex={1}>
          <Request.Skeleton count={3} opacity={0.5} />
        </Stack>
      )) ||
        (requests.size > 0 && (
          <>
            <ScrollArea
              flex={1}
              scrollbarSize={4}
              type="scroll"
              scrollbars="y"
              offsetScrollbars="y"
              pr={2}
              mr={-(4 + 2)}
              c={ThemeColor.NavbarText}
              onClick={handleOutsideClick}
            >
              <AnimatedStack gap="xs">
                {Array.from(requests, ([id, request]) => (
                  <AnimatedStack.Item key={id} data-request-item>
                    <Request request={request} active={request.id === rID} />
                  </AnimatedStack.Item>
                ))}
              </AnimatedStack>
            </ScrollArea>
            {requests.size > 1 && <DeleteAllButton />}
          </>
        )) || <NoRequestsYet flex={1} />}

      <Stack w="100%" gap="xs" c={ThemeColor.NavbarText}>
        <LangSelect />
      </Stack>
    </Stack>
  )
}

const NoRequestsYet = ({ ...props }: BoxProps): React.JSX.Element => {
  const isDark = useComputedColorScheme() === 'dark'
  const { t } = useL10n()

  return (
    <Box {...props}>
      <Stack align="center" justify="center" h="100%" opacity={isDark ? 0.4 : 0.8}>
        <Text fz="xs" c="white" ta="center">
          {t(L10nKey.noRequestsCapturedYet)}. {t(L10nKey.sendFirstRequestToSee)}.
        </Text>
        <Loader type="dots" color="white" size="1em" mr={8} mb={3} opacity={0.75} />
      </Stack>
    </Box>
  )
}
