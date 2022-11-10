import {defineComponent} from 'vue'
import {NavigationFailure, RouteLocationNormalized} from 'vue-router'

interface RouteParts {
  sessionUUID: string | undefined
  requestUUID: string | undefined
}

export default defineComponent({
  methods: {
    navigateToIndex(
      sessionUUID: string | undefined | null,
      requestUUID: string | undefined | null,
    ): Promise<void | NavigationFailure | undefined> {
      const params = this.getFromRoute(this.$router.currentRoute.value)

      if (sessionUUID !== null) {
        params.sessionUUID = sessionUUID
      }

      if (requestUUID !== null) {
        params.requestUUID = requestUUID
      }

      return this.$router.replace({
        name: 'request',
        params: {
          sessionUUID: params.sessionUUID,
          requestUUID: params.requestUUID,
        },
      })
    },
  },
})
