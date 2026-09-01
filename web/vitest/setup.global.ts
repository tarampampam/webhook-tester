import { mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const LOCALSTORAGE_DIR = join(tmpdir(), `vitest-localstorage-${process.pid}`)
const LOCALSTORAGE_FILE = join(LOCALSTORAGE_DIR, 'storage.data')

/**
 * This is the workaround for: https://github.com/vitest-dev/vitest/issues/8757
 *
 * Without this you will get "`--localstorage-file` was provided without a valid path" warning.
 */
export default async function setup() {
  await mkdir(LOCALSTORAGE_DIR, { recursive: true })

  const existing = process.env.NODE_OPTIONS || ''
  process.env.NODE_OPTIONS = `${existing} --localstorage-file=${LOCALSTORAGE_FILE}`.trim()

  return async function teardown() {
    try {
      await rm(LOCALSTORAGE_DIR, { recursive: true, force: true })
    } catch (e) {
      console.warn('Failed to clean up local storage directory:', e)
    }
  }
}
