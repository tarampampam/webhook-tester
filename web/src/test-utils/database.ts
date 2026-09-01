import { IDBFactory, IDBKeyRange } from 'fake-indexeddb'
import { Database } from '~/db'

export const newTestDatabase = (): Database => new Database({ indexedDB: new IDBFactory(), IDBKeyRange })
