import { Burger, Group } from '@mantine/core'
import React from 'react'

export const Header = ({ opened, onToggle }: { opened: boolean; onToggle: () => void }): React.JSX.Element => (
  <Group h="100%" px="md">
    <Burger opened={opened} onClick={onToggle} hiddenFrom="sm" size="sm" />
  </Group>
)
