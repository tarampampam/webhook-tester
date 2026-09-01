import type { Table } from 'dexie'

export type SessionBody = {
  id: string
  body: Uint8Array
}

export type SessionBodiesTable = Table<SessionBody, string>

export const sessionBodiesSchema = {
  sessionBodies: '&id',
}
