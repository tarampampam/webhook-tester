import type { Table } from 'dexie'

export type Session = {
  id: string
  response: {
    code: number
    headers: Array<{ name: string; value: string }>
    delay: number
  }
  createdAt: Date
}

export type SessionsTable = Table<Session, string>

export const sessionsSchema = {
  sessions: '&id, createdAt',
}
