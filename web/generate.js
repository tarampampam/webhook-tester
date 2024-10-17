#!/usr/bin/env node

import fs from 'node:fs'
import process from 'node:process'
import openapiTS, { astToString } from 'openapi-typescript'

/** @param {string} message */
const panic = (message) => {
  console.error(message)
  process.exit(1)
}

/**
 * @param {string} input Source OpenAPI file
 * @param {string} output Output d.ts file
 * @return {Promise<void>}
 */
const generate = async (input, output) => {
  const ast = await openapiTS(new URL(input, import.meta.url), {
    additionalProperties: false,
    arrayLength: true,
    emptyObjectsUnknown: false,
    enum: true,
    immutable: true,
  })

  fs.writeFileSync(output, astToString(ast))
}

const output = process.argv[3]
const input = process.argv[2]

if (!input || typeof input !== 'string') {
  panic('Please provide an input file')
} else if (!output || typeof output !== 'string') {
  panic('Please provide an output file')
}

await Promise.all([generate(input, output)]).catch((error) => {
  panic(error)
})
