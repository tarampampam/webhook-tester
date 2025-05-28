import { Table } from 'dexie'

export type Session = {
  sID: string
  responseCode: number
  responseHeaders: Array<{ name: string; value: string }>
  responseDelay: number
  responseBody: Uint8Array
  createdAt: Date
  proxyUrls?: string[]
  proxyResponseMode?: string
}

export type SessionsTable = Table<Session, string>

export const sessionsSchema = {
  sessions: '&sID',
}
