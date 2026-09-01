import { Dexie, type DexieOptions } from 'dexie'
import { type RequestPayload, requestPayloadsSchema, type RequestPayloadsTable } from './tables/request-payloads'
import { type Request, requestsSchema, type RequestsTable } from './tables/requests'
import { sessionBodiesSchema, type SessionBodiesTable, type SessionBody } from './tables/session-bodies'
import { type Session, sessionsSchema, type SessionsTable } from './tables/sessions'

/** Input type for putSession. body is stored in a separate table; omit or pass empty to clear. */
export type SessionInput = Omit<Session, 'response'> & {
  response: Session['response'] & { body?: Uint8Array }
}

/** Session metadata with lazy body loader. */
type SessionRecord = Omit<Session, 'response'> & {
  response: Session['response'] & { getBody(): Promise<Uint8Array | null> }
}

/** Input type for putRequest. payload is stored in a separate table; omit to clear. */
export type RequestInput = Request & { payload?: Uint8Array }

/** Request metadata with lazy payload loader. */
type RequestRecord = Request & {
  getPayload(): Promise<Uint8Array | null>
}

export class Database {
  private readonly dexie: Dexie
  private readonly sessions: SessionsTable
  private readonly sessionBodies: SessionBodiesTable
  private readonly requests: RequestsTable
  private readonly requestPayloads: RequestPayloadsTable

  constructor(options?: DexieOptions) {
    this.dexie = new Dexie('wht-v3-db', options) // https://dexie.org/docs/Typescript
    this.dexie.version(1).stores({
      ...sessionsSchema,
      ...sessionBodiesSchema,
      ...requestsSchema,
      ...requestPayloadsSchema,
    })

    this.sessions = this.dexie.table('sessions')
    this.sessionBodies = this.dexie.table('sessionBodies')
    this.requests = this.dexie.table('requests')
    this.requestPayloads = this.dexie.table('requestPayloads')
  }

  /**
   * Insert a new session (the existing session with the same id will be replaced).
   * A non-empty body is upserted into sessionBodies; undefined or empty deletes any existing entry.
   */
  async putSession(...data: Array<SessionInput>): Promise<void> {
    if (data.length === 0) {
      return
    }

    await this.dexie.transaction('rw', this.sessions, this.sessionBodies, async () => {
      const sessions: Array<Session> = []
      const bodiesToInsert: Array<SessionBody> = []
      const bodyIDsToDelete: Array<string> = []

      for (const item of data) {
        const { body, ...responseMeta } = item.response
        sessions.push({ ...item, response: responseMeta })

        if (body !== undefined) {
          bodiesToInsert.push({ id: item.id, body })
        } else {
          bodyIDsToDelete.push(item.id)
        }
      }

      await this.sessions.bulkPut(sessions)

      if (bodiesToInsert.length > 0) {
        await this.sessionBodies.bulkPut(bodiesToInsert)
      }

      if (bodyIDsToDelete.length > 0) {
        await this.sessionBodies.bulkDelete(bodyIDsToDelete)
      }
    })
  }

  /**
   * Get all available session IDs, ordered by creation date from the newest to the oldest.
   */
  async getSessionIDs(): Promise<Array<string>> {
    return this.sessions.orderBy('createdAt').reverse().primaryKeys()
  }

  /**
   * Get the session by id. Returns null if not found.
   */
  async getSession(id: string): Promise<SessionRecord | null> {
    const data = await this.sessions.get(id)
    if (!data) {
      return null
    }

    return {
      ...data,
      response: {
        ...data.response,
        getBody: async (): Promise<Uint8Array | null> => {
          const record = await this.sessionBodies.get(id)
          return record?.body ?? null
        },
      },
    }
  }

  /**
   * Delete sessions by id, cascading to their bodies, associated requests, and request payloads.
   */
  async deleteSession(...sIDs: Array<string>): Promise<void> {
    if (sIDs.length === 0) {
      return
    }

    await this.dexie.transaction(
      'rw',
      this.sessions,
      this.sessionBodies,
      this.requests,
      this.requestPayloads,
      async () => {
        const rIDs = (await this.requests.where('sID').anyOf(sIDs).toArray()).map((r) => r.rID)

        await this.sessions.bulkDelete(sIDs)
        await this.sessionBodies.bulkDelete(sIDs)
        await this.requests.where('sID').anyOf(sIDs).delete()

        if (rIDs.length > 0) {
          await this.requestPayloads.bulkDelete(rIDs)
        }
      }
    )
  }

  /**
   * Insert new requests (any existing request with the same rID will be replaced).
   *
   * If a limit is provided, the total number of stored requests for the session will be restricted to the most
   * recent `limit` entries after insertion. Older entries will be deleted.
   */
  async putRequests(data: Array<RequestInput>, limit?: number): Promise<void> {
    if (data.length === 0) {
      return
    }

    await this.dexie.transaction('rw', this.requests, this.requestPayloads, async () => {
      const requests: Array<Request> = []
      const payloadsToInsert: Array<RequestPayload> = []
      const payloadsToDelete: Array<string> = []

      for (const item of data) {
        const { payload, ...row } = item
        requests.push(row)

        if (payload !== undefined) {
          payloadsToInsert.push({ id: item.rID, payload })
        } else {
          payloadsToDelete.push(item.rID)
        }
      }

      await this.requests.bulkPut(requests)

      if (payloadsToInsert.length > 0) {
        await this.requestPayloads.bulkPut(payloadsToInsert)
      }

      if (payloadsToDelete.length > 0) {
        await this.requestPayloads.bulkDelete(payloadsToDelete)
      }

      if (limit !== undefined && limit > 0) {
        for (const sID of new Set(data.map((r) => r.sID))) {
          await this.limitRequestsCount(sID, limit)
        }
      }
    })
  }

  /**
   * Delete the oldest requests for a session, keeping only the most recent `limit` entries.
   */
  async trimRequests(sID: string, limit: number): Promise<void> {
    if (limit <= 0) {
      return
    }

    await this.dexie.transaction('rw', this.requests, this.requestPayloads, async () =>
      this.limitRequestsCount(sID, limit)
    )
  }

  /**
   * ⚠ Must be used inside a transaction.
   *
   * @example
   * ```ts
   * await this.dexie.transaction('rw', this.requests, this.requestPayloads, async () =>
   *   this.limitRequestsCount(sID, limit)
   * )
   * ```
   */
  private async limitRequestsCount(sID: string, limit: number): Promise<void> {
    if (limit <= 0) {
      return // just in case
    }

    const overflow = await this.requests
      .where('[sID+capturedAt]')
      .between([sID, Dexie.minKey], [sID, Dexie.maxKey])
      .reverse()
      .offset(limit)
      .primaryKeys()

    if (overflow.length > 0) {
      await this.requests.bulkDelete(overflow)
      await this.requestPayloads.bulkDelete(overflow)
    }
  }

  /**
   * Insert or update a request payload by rID. The request record with the rID must already exist.
   */
  async putRequestPayload(rID: string, payload: Uint8Array): Promise<void> {
    await this.requestPayloads.put({ id: rID, payload })
  }

  /**
   * Get a request by rID. Returns null if not found.
   */
  async getRequest(rID: string): Promise<RequestRecord | null> {
    const data = await this.requests.get(rID)
    if (!data) {
      return null
    }

    return {
      ...data,
      getPayload: async (): Promise<Uint8Array | null> => {
        const record = await this.requestPayloads.get(rID)
        return record?.payload ?? null
      },
    }
  }

  /**
   * Get all session requests, ordered by creation date from the newest to the oldest.
   */
  async getRequests(sID: string): Promise<Array<RequestRecord>> {
    const rows = await this.requests
      .where('[sID+capturedAt]')
      .between([sID, Dexie.minKey], [sID, Dexie.maxKey])
      .reverse()
      .toArray()

    return rows.map((row) => ({
      ...row,
      getPayload: this.newRequestPayloadReader(row.rID),
    }))
  }

  /**
   * Get a lazy payload loader for a request by rID.
   */
  newRequestPayloadReader(rID: string): () => Promise<Uint8Array | null> {
    return async (): Promise<Uint8Array | null> => {
      const record = await this.requestPayloads.get(rID)
      return record?.payload ?? null
    }
  }

  /**
   * Delete requests by rID, cascading to their payloads.
   */
  async deleteRequest(...rIDs: Array<string>): Promise<void> {
    if (rIDs.length === 0) {
      return
    }

    await this.dexie.transaction('rw', this.requests, this.requestPayloads, async () => {
      await this.requests.bulkDelete(rIDs)
      await this.requestPayloads.bulkDelete(rIDs)
    })
  }

  /**
   * Delete all requests associated with a session, cascading to their payloads.
   */
  async deleteAllRequests(sID: string): Promise<void> {
    await this.dexie.transaction('rw', this.requests, this.requestPayloads, async () => {
      const rIDs = (await this.requests.where('sID').equals(sID).toArray()).map((r) => r.rID)

      await this.requests.where('sID').equals(sID).delete()

      if (rIDs.length > 0) {
        await this.requestPayloads.bulkDelete(rIDs)
      }
    })
  }
}
