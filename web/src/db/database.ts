import { Dexie } from 'dexie'
import { DatabaseError } from './errors'
import { SessionsTable, Session, sessionsSchema, RequestsTable, Request, requestsSchema } from './tables'

export class Database {
  public dexie: Dexie
  private readonly sessions: SessionsTable
  private readonly requests: RequestsTable

  constructor() {
    // create database
    this.dexie = new Dexie('webhook-tester-v2-db') // https://dexie.org/docs/Typescript
    this.dexie.version(1).stores({ ...sessionsSchema, ...requestsSchema })

    // assign tables
    this.sessions = this.dexie.table('sessions')
    this.requests = this.dexie.table('requests')
  }

  /**
   * Insert a new session (the existing session with the same sID will be replaced).
   *
   * @throws {DatabaseError} If the operation fails
   */
  async createSession(...data: Array<Session>): Promise<void> {
    try {
      await this.dexie.transaction('rw', this.sessions, async () => {
        await this.sessions.bulkPut(data)
      })
    } catch (err) {
      throw new DatabaseError('Failed to create session', err)
    }
  }

  /**
   * Get all available session IDs, ordered by creation date from the newest to the oldest.
   *
   * @throws {DatabaseError} If the operation fails
   */
  async getSessionIDs(): Promise<Array<string>> {
    try {
      return await this.dexie.transaction('r', this.sessions, async () => {
        return (await this.sessions.toCollection().sortBy('createdAt')).reverse().map((session) => session.sID)
      })
    } catch (err) {
      throw new DatabaseError('Failed to get session IDs', err)
    }
  }

  /**
   * Get the session by sID.
   *
   * @throws {DatabaseError} If the operation fails
   */
  async getSession(sID: string): Promise<Session | null> {
    try {
      return await this.dexie.transaction('r', this.sessions, async () => {
        return (await this.sessions.get(sID)) || null
      })
    } catch (err) {
      throw new DatabaseError('Failed to get session', err)
    }
  }

  /**
   * Get many sessions by its sID.
   *
   * @throws {DatabaseError} If the operation fails
   */
  async getSessions<T extends string>(...sID: Array<T>): Promise<{ [K in T]: Session | null }> {
    try {
      return await this.dexie.transaction('r', this.sessions, async () => {
        const sessions = await this.sessions.where('sID').anyOf(sID).toArray()

        return sID.reduce(
          (acc, sID_1) => {
            acc[sID_1] = sessions.find((session) => session.sID === sID_1) || null

            return acc
          },
          {} as {
            [K in T]: Session | null
          }
        )
      })
    } catch (err) {
      throw new DatabaseError('Failed to get sessions', err)
    }
  }

  /**
   * Check if a session exists.
   *
   * @throws {DatabaseError} If the operation fails
   */
  async sessionExists(sID: string): Promise<boolean> {
    try {
      return await this.dexie.transaction('r', this.sessions, async () => {
        return (await this.sessions.where('sID').equals(sID).count()) > 0
      })
    } catch (err) {
      throw new DatabaseError('Failed to check if session exists', err)
    }
  }

  /**
   * Get all session requests, ordered by creation date from the newest to the oldest.
   *
   * @throws {DatabaseError} If the operation fails
   */
  async getSessionRequests(sID: string): Promise<Array<Request>> {
    try {
      return await this.dexie.transaction('r', this.requests, async () => {
        return (await this.requests.where('sID').equals(sID).sortBy('capturedAt')).reverse()
      })
    } catch (err) {
      throw new DatabaseError('Failed to get session requests', err)
    }
  }

  /**
   * Delete session (and all requests associated with it).
   *
   * @throws {DatabaseError} If the operation fails
   */
  async deleteSession(...sID: Array<string>): Promise<void> {
    try {
      await this.dexie.transaction('rw', this.sessions, this.requests, async () => {
        await this.sessions.bulkDelete(sID)
        await this.requests.where('sID').anyOf(sID).delete()
      })
    } catch (err) {
      throw new DatabaseError('Failed to delete session', err)
    }
  }

  /**
   * Insert a new request (the existing request with the same rID will be replaced).
   *
   * @throws {DatabaseError} If the operation fails
   */
  async createRequest(...data: Array<Request>): Promise<void> {
    try {
      await this.dexie.transaction('rw', this.requests, async () => {
        await this.requests.bulkPut(data)
      })
    } catch (err) {
      throw new DatabaseError('Failed to create request', err)
    }
  }

  /**
   * Get a request by rID.
   *
   * @throws {DatabaseError} If the operation fails
   */
  async getRequest(rID: string): Promise<Request | null> {
    try {
      return await this.dexie.transaction('r', this.requests, async () => {
        return (await this.requests.get(rID)) || null
      })
    } catch (err) {
      throw new DatabaseError('Failed to get request', err)
    }
  }

  /**
   * Delete requests by rID.
   *
   * @throws {DatabaseError} If the operation fails
   */
  async deleteRequest(...rID: Array<string>): Promise<void> {
    try {
      await this.dexie.transaction('rw', this.requests, async () => {
        await this.requests.bulkDelete(rID)
      })
    } catch (err) {
      throw new DatabaseError('Failed to delete request', err)
    }
  }

  /**
   * Delete all requests associated with a session.
   *
   * @throws {DatabaseError} If the operation fails
   */
  async deleteAllRequests(sID: string): Promise<void> {
    try {
      await this.dexie.transaction('rw', this.requests, async () => {
        await this.requests.where('sID').equals(sID).delete()
      })
    } catch (err) {
      throw new DatabaseError('Failed to delete all requests', err)
    }
  }
}
