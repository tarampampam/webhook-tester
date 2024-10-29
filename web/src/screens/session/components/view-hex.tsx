import { Alert, Box, Divider, Grid, NativeSelect } from '@mantine/core'
import { IconInfoCircle, IconScissors } from '@tabler/icons-react'
import React, { useEffect, useState } from 'react'

type DataItem = [string /* representation */, string | undefined /* ascii */]
type DataLine = Array<DataItem>

export default function ViewHex({
  input,
  lengthLimit = 1024 * 24, // 24KB
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

  const fontSize: string | number = '0.75em'

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
          <Box style={{ fontFamily: 'monospace', whiteSpace: 'pre', textAlign: 'right', fontSize }} c="dimmed">
            {!!dataItems && // print line numbers
              '\n' +
                Array<number>(dataItems.length)
                  .fill(0)
                  .map((_, lineNum) => byteToString(lineNum * lineSize, lineNumberType).toUpperCase())
                  .join('\n')}
          </Box>
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
          <Box style={{ fontFamily: 'monospace', whiteSpace: 'pre', fontSize }}>
            <Box component="span" c="dimmed" size={fontSize}>
              {!!dataItems && // print column numbers
                dataItems.length &&
                dataItems[0]
                  .map((_, colNum) => byteToString(colNum, displayType).toUpperCase() + (colNum % 4 === 3 ? ' ' : ''))
                  .join(' ')
                  .trimEnd()}
            </Box>
            {!!dataItems &&
              '\n' +
                dataItems
                  .map((line) =>
                    line
                      .map(([byte], colNum) => byte + (colNum % 4 === 3 ? ' ' : ''))
                      .join(' ')
                      .trimEnd()
                  )
                  .join('\n')}
            {trimmed && (
              <Divider
                my="xs"
                label={
                  <>
                    <IconScissors size="1.2em" />
                    <Box mx="xs">Data trimmed</Box>
                    <IconScissors size="1.2em" />
                  </>
                }
                labelPosition="center"
              />
            )}
          </Box>
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
          <Box style={{ fontFamily: 'monospace', whiteSpace: 'pre', fontSize }}>
            {!!dataItems &&
              '\n' +
                dataItems
                  .map((line) => line.map(([, ascii]) => (typeof ascii === 'string' ? ascii : '·')).join(''))
                  .join('\n')}
          </Box>
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

const LineSizes: ReadonlyArray<number> = [4, 8, 16, 32]

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
  return byte >= 32 && byte <= 126 ? String.fromCharCode(byte) : undefined
}
