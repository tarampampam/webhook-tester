import { Table } from 'dexie'

export type Session = {
  sID: string
  humanReadableName: string
  createdAt: Date
}

export type SessionsTable = Table<Session, string>

export const sessionsSchema = {
  sessions: '&sID',
}
