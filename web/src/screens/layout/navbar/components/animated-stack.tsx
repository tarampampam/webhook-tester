import { Stack } from '@mantine/core'
import type { HTMLMotionProps } from 'motion/react'
import { AnimatePresence, motion } from 'motion/react'
import React from 'react'

const MotionStack = motion.create(Stack)

export const AnimatedStack = Object.assign(
  ({
    children,
    ...props
  }: { children: React.ReactNode } & React.ComponentPropsWithoutRef<typeof MotionStack>): React.JSX.Element => (
    <MotionStack layout style={{ position: 'relative' }} {...props}>
      <AnimatePresence initial={false} mode="popLayout">
        {children}
      </AnimatePresence>
    </MotionStack>
  ),
  {
    Item: ({
      children,
      ref,
      ...props
    }: {
      children: React.ReactNode
      ref?: React.Ref<HTMLDivElement>
    } & HTMLMotionProps<'div'>): React.JSX.Element => (
      <motion.div
        ref={ref}
        initial={{ opacity: 0, scale: 0.92, y: -6 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.92, y: -6 }}
        transition={{
          duration: 0.18,
          ease: 'easeInOut',
          layout: { type: 'spring', stiffness: 400, damping: 30 },
        }}
        layout
        {...props}
      >
        {children}
      </motion.div>
    ),
  }
)
