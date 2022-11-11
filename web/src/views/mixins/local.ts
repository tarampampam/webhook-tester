import {defineComponent} from 'vue'
import {isValidUUID} from '../../utils'

const keys = {
  session: 'session-uuid',
  autoNavigate: 'auto-navigate',
  showDetails: 'show-details',
}

type booleanKeys = 'auto-navigate' | 'show-details'

const boolValues = {yes: 'yes', no: 'no'}

export default defineComponent({
  methods: {
    getLocalSessionUUID(): string | undefined {
      const value = localStorage.getItem(keys.session)

      if (value && isValidUUID(value)) {
        return value
      }

      return undefined
    },

    setLocalSessionUUID(uuid: string): void {
      if (isValidUUID(uuid)) {
        localStorage.setItem(keys.session, uuid)
      } else {
        throw new Error(`invalid session UUID (${uuid}) cannot be stored`)
      }
    },

    removeLocalSessionUUID(): void {
      localStorage.removeItem(keys.session)
    },

    setLocalBool(name: booleanKeys, value: boolean): void {
      let key: string

      switch (name) {
        case 'auto-navigate':
          key = keys.autoNavigate
          break

        case 'show-details':
          key = keys.showDetails
          break

        default:
          return
      }

      localStorage.setItem(key, value ? boolValues.yes : boolValues.no)
    },

    getLocalBool(name: booleanKeys, def: boolean): boolean {
      let key: string

      switch (name) {
        case 'auto-navigate':
          key = keys.autoNavigate
          break

        case 'show-details':
          key = keys.showDetails
          break

        default:
          return def
      }

      const value = localStorage.getItem(key)

      if (value) {
        switch (value) {
          case boolValues.yes:
            return true

          case boolValues.no:
            return false
        }
      }

      return def
    },
  },
})
