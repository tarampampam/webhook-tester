import { Button, type ButtonProps } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconTrash } from '@tabler/icons-react'
import React, { useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { pathTo, ROUTE_ID, useActiveSessionID } from '~/routing'
import { anyToError, L10nKey, useL10n, useRequests } from '~/shared'

export const DeleteAllButton = ({ ...props }: ButtonProps): React.JSX.Element => {
  const { delAllRequests } = useRequests()
  const sID = useActiveSessionID()
  const navigate = useNavigate()
  const { t } = useL10n()

  const handleClick = useCallback((): void => {
    if (sID) {
      delAllRequests(sID).catch((err) => {
        notifications.show({
          title: t(L10nKey.somethingWentWrong),
          message: anyToError(err).message,
          color: 'red',
        })
      })

      navigate(pathTo(ROUTE_ID.Session, { sID }), { replace: true })
    }
  }, [delAllRequests, navigate, sID, t])

  return (
    <Button
      size="compact-xs"
      variant="default"
      leftSection={<IconTrash size="1em" />}
      fw="inherit"
      fz="0.75em"
      onClick={handleClick}
      fullWidth
      {...props}
    >
      {t(L10nKey.deleteAllRequests)}
    </Button>
  )
}
