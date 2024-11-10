import { Table } from 'dexie'

export type Request = {
  sID: string
  rID: string
  clientAddress: string
  method: string
  headers: Array<{ name: string; value: string }>
  url: URL
  capturedAt: Date
}

export type RequestsTable = Table<Request, string>

export const requestsSchema = {
  requests: '&rID, sID',
}
