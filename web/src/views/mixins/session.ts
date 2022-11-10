import {defineComponent} from 'vue'
import {NavigationFailure} from 'vue-router'
import {isValidUUID} from "../../utils";

const storageSessionUuidKey = 'session_uuid_v2'

export default defineComponent({
  methods: {
    getLocalSessionUUID(): string | undefined {
      const value = localStorage.getItem(storageSessionUuidKey)

      if (value && isValidUUID(value)) {
        return value
      }

      return undefined
    },

    setLocalSessionUUID(uuid: string): void {
      if (isValidUUID(uuid)) {
        localStorage.setItem(storageSessionUuidKey, uuid)
      } else {
        throw new Error(`invalid session UUID (${uuid}) cannot be stored`)
      }
    },

    removeLocalSessionUUID(): void {
      localStorage.removeItem(storageSessionUuidKey)
    }
  },
})
