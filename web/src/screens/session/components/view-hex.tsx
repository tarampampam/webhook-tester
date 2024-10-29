import { Grid, Group, NativeSelect, Stack, Alert } from '@mantine/core'
import React, { useEffect, useState } from 'react'
import { IconInfoCircle } from '@tabler/icons-react'

type DataItem = [string /* representation */, string | undefined /* ascii */]
type DataLine = Array<DataItem>

export default function ViewHex({
  input,
  lengthLimit = 1024 * 10, // 10KB
}: {
  input: Uint8Array
  lengthLimit?: number
}): React.JSX.Element {
  const [lineNumberType, setLineNumberType] = useState<NumberBase>(NumberBase.Hexadecimal)
  const [displayType, setDisplayType] = useState<NumberBase>(NumberBase.Hexadecimal)
  const [lineSize, setLineSize] = useState<number>(16)
  const [trimmed, setTrimmed] = useState<boolean>(false)

  /**
   * Data items are stored in a 2D array where each item is a tuple of two strings:
   *  - the first string is the byte representation in hexadecimal, octal, binary, or decimal format
   *  - the second string is the ASCII representation of the byte
   *
   * The data items are grouped by lines where each line contains a fixed number of items
   * determined by the `lineSize` state.
   *
   * Unprintable characters are represented as undefined in the ASCII representation, but they are
   * still included in the data items array.
   *
   * @example
   * [
   *   [['68', 'h'], ['65', 'e'], ['6c', 'l'], ['6c', 'l'], ['6f', 'o']],
   *   [['20', ' '], ['77', 'w'], ['6f', 'o'], ['72', 'r'], ['6c', 'l']],
   *   [['64', 'd'], ['0a', undefined]]
   * ]
   */
  const [dataItems, setDataItems] = useState<ReadonlyArray<DataLine>>([])

  useEffect(() => {
    if (input.length === 0) {
      setDataItems([])

      return
    }

    const length = input.length > lengthLimit ? lengthLimit : input.length // limit the number of bytes
    setTrimmed(input.length > lengthLimit)

    // preallocate the data items array
    const lines = new Array<DataLine>(Math.ceil(length / lineSize))

    // fill the data items array with empty items
    for (let lineNum = 0; lineNum < lines.length; lineNum++) {
      const start = lineNum * lineSize

      lines[lineNum] = new Array<DataItem>(Math.min(start + lineSize, length) - start).fill(['00', undefined])
    }

    // fill the data items array with the actual data, handling all the heavy lifting here
    for (let lineNum = 0; lineNum < lines.length; lineNum++) {
      for (let itemNum = 0; itemNum < lines[lineNum].length; itemNum++) {
        const byte = input[lineNum * lineSize + itemNum]

        lines[lineNum][itemNum] = [byteToString(byte, displayType), byteToAscii(byte)] satisfies DataItem
      }
    }

    setDataItems(lines)
  }, [input, displayType, lineSize, lengthLimit])

  return (
    <>
      {trimmed && (
        <Alert color="yellow" my="sm" title="Data trimmed" icon={<IconInfoCircle />}>
          The request body is large and has been trimmed to {lengthLimit} bytes for performance reasons.
        </Alert>
      )}
      <Grid my="sm">
        <Grid.Col span="content" visibleFrom="md">
          <NativeSelect
            size="xs"
            mb="sm"
            description="Line number type"
            ta="right"
            data={Object.values(NumberBase)}
            value={lineNumberType}
            onChange={(e) => setLineNumberType(e.currentTarget.value as NumberBase)}
          />
          <Stack style={{ fontFamily: 'monospace' }}>
            <Group c="dimmed">&nbsp;</Group>
            {!!dataItems && // print line numbers
              Array(dataItems.length)
                .fill(0)
                .map((_, lineNum) => (
                  <Group justify="flex-end" key={lineNum} c="dimmed">
                    {byteToString(lineNum * lineSize, lineNumberType)}
                  </Group>
                ))}
          </Stack>
        </Grid.Col>
        <Grid.Col span="content" mx="lg">
          <NativeSelect
            size="xs"
            mb="sm"
            description="Data representation type"
            data={Object.values(NumberBase)}
            value={displayType}
            onChange={(e) => setDisplayType(e.currentTarget.value as NumberBase)}
          />
          <Stack style={{ fontFamily: 'monospace' }}>
            <Group gap="0.3em">
              {!!dataItems && // print column numbers
                dataItems.length &&
                dataItems[0].map((_, itemNum) => {
                  return (
                    <Group key={itemNum} c="dimmed">
                      <div>{byteToString(itemNum, displayType).toUpperCase()}</div>
                    </Group>
                  )
                })}
            </Group>
            {!!dataItems &&
              dataItems.map((line, lineNum) => {
                return (
                  <Group gap="0.3em" key={lineNum}>
                    {line.map(([byte], itemNum) => {
                      return (
                        <Group key={lineNum + itemNum}>
                          <div>{byte}</div>
                        </Group>
                      )
                    })}
                  </Group>
                )
              })}
          </Stack>
        </Grid.Col>
        <Grid.Col span="auto">
          <NativeSelect
            size="xs"
            mb="sm"
            description="Line size in bytes"
            data={LineSizes.map((size) => size.toString())}
            value={lineSize.toString()}
            onChange={(e) => setLineSize(parseInt(e.currentTarget.value))}
          />
          <Stack style={{ fontFamily: 'monospace' }}>
            <Group c="dimmed">&nbsp;</Group>
            {!!dataItems &&
              dataItems.map((line, lineNum) => {
                return (
                  <Group gap="0.1em" key={lineNum}>
                    {line.map(([, ascii], itemNum) => {
                      return (
                        <Group key={itemNum}>
                          <div>_{typeof ascii === 'string' ? ascii : '·'}</div>
                        </Group>
                      )
                    })}
                  </Group>
                )
              })}
          </Stack>
        </Grid.Col>
      </Grid>
    </>
  )
}

enum NumberBase { // hello
  Binary = 'Binary', // 01101000, 01100101, 01101100, 01101100, 01101111
  Octal = 'Octal', // 150, 145, 154, 154, 157
  Decimal = 'Decimal', // 104, 101, 108, 108, 111
  Hexadecimal = 'Hexadecimal', // 68, 65, 6c, 6c, 6f
}

const LineSizes: ReadonlyArray<number> = [1, 2, 4, 8, 16, 32]

const byteToString = (byte: number, base: NumberBase): string => {
  switch (base) {
    case NumberBase.Binary:
      return byte.toString(2).padStart(8, '0')

    case NumberBase.Octal:
      return byte.toString(8).padStart(3, '0')

    case NumberBase.Decimal:
      return byte.toString(10).padStart(3, '0')

    case NumberBase.Hexadecimal:
      return byte.toString(16).padStart(2, '0')

    default:
      throw new Error('Invalid data representation type')
  }
}

const byteToAscii = (byte: number): string | undefined => {
  if (byte >= 32 && byte <= 126) {
    return String.fromCharCode(byte)
  }

  return undefined
}
