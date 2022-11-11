import {defineComponent} from 'vue'
import {NavigationFailure, Router} from 'vue-router'

type NavigationResult = Promise<void | NavigationFailure | undefined>

export default defineComponent({
  methods: {
    navigateToIndex(router: Router): NavigationResult {
      return router.push({name: 'index'})
    },

    navigateToSession(router: Router, uuid: string): NavigationResult {
      return router.push({
        name: 'request',
        params: {
          sessionUUID: uuid,
          requestUUID: undefined,
        },
      })
    },

    navigateToRequest(router: Router, sessionUUID: string, uuid: string): NavigationResult {
      return router.push({
        name: 'request',
        params: {
          sessionUUID: sessionUUID,
          requestUUID: uuid,
        },
      })
    },
  },
})
