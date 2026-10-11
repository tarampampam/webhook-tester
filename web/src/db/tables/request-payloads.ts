import type { Table } from 'dexie'

export type RequestPayload = {
  id: string // rID
  payload: Uint8Array
}

export type RequestPayloadsTable = Table<RequestPayload, string>

export const requestPayloadsSchema = {
  requestPayloads: '&id',
}
