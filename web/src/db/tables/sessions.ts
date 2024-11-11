import { Table } from 'dexie'

export type Session = {
  sID: string
  humanReadableName: string
  responseCode: number
  responseHeaders: Array<{ name: string; value: string }>
  responseDelay: number
  responseBody: Uint8Array
  createdAt: Date
}

export type SessionsTable = Table<Session, string>

export const sessionsSchema = {
  sessions: '&sID',
}
