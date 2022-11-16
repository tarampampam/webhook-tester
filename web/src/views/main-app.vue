<template>
  <main>
    <main-header
      :current-web-hook-url="currentWebHookUrl"
      :session-lifetime-sec="sessionLifetimeSec"
      :max-body-size-bytes="maxBodySize"
      :version="appVersion"
      @createNewUrl="startNewSession"
    />

    <div class="container-fluid mb-2">
      <div class="row flex-xl-nowrap">
        <div
          class="sidebar px-2 py-0"
          @click.self="switchToRequest(undefined)"
        >
          <div class="ps-3 pt-4 pe-3 pb-3">
            <div class="d-flex w-100 justify-content-between">
              <h5 class="text-uppercase mb-0">
                Requests
                <span class="badge bg-primary rounded-pill ms-1 total-requests-count">{{ requests.length }}</span>
              </h5>
              <button
                type="button"
                class="btn btn-outline-danger btn-sm position-relative button-delete-all"
                v-if="requests.length > 0"
                @click="deleteAllRequests(true)"
              >
                Delete all
              </button>
            </div>
          </div>

          <div
            class="list-group"
            v-if="requests.length > 0"
          >
            <request-plate
              v-for="r in requests"
              :key="r.UUID"
              :request="r"
              :class="{ active: requestUUID === r.UUID }"
              @click="switchToRequest(r.UUID)"
              @onDelete="(uuid: string) => deleteRequest(uuid, true)"
            />
          </div>
          <div
            v-else
            class="text-muted text-center mt-3"
          >
            <span class="spinner-border spinner-border-sm me-1" /> Waiting for first request
          </div>
        </div>

        <div
          class="col py-3 ps-md-4"
          role="main"
        >
          <div v-if="requests.length > 0 && requestExist(requestUUID)">
            <div class="row pt-2">
              <requests-navigator
                class="col-6"
                :requests="requests"
                :request-u-u-i-d="requestUUID"
                @changed="(uuid: string) => switchToRequest(uuid)"
              />
              <div class="col-6 pb-1 text-end">
                <div class="form-check d-inline-block">
                  <input
                    type="checkbox"
                    class="form-check-input"
                    id="show-details"
                    v-model="showRequestDetails"
                  >
                  <label
                    class="form-check-label"
                    for="show-details"
                  >Show details</label>
                </div>
                <div
                  class="form-check d-inline-block ms-3"
                  title="Automatically select and go to the latest incoming webhook request"
                >
                  <input
                    type="checkbox"
                    class="form-check-input"
                    id="auto-navigate"
                    v-model="autoRequestNavigate"
                  >
                  <label
                    class="form-check-label"
                    for="auto-navigate"
                  >Auto navigate</label>
                </div>
              </div>
            </div>

            <request-details
              v-if="showRequestDetails"
              class="pt-3"
              :request="request()"
            />

            <div class="pt-3">
              <h4>Request body</h4>

              <request-body
                v-if="request() && request().content.length"
                :request="request()"
              />
              <div
                v-else
                class="pt-1 pb-1"
              >
                <p class="text-muted small text-monospace">
                  // empty request body
                </p>
              </div>
            </div>
          </div>
          <index-empty
            v-else
            :current-web-hook-url="currentWebHookUrl"
          />
        </div>
      </div>
    </div>
  </main>
</template>

<script lang="ts">
import {defineComponent} from 'vue'
import IndexEmpty from './components/index-empty.vue'
import MainHeader from './components/main-header.vue'
import RequestPlate from './components/request-plate.vue'
import RequestsNavigator from './components/requests-navigator.vue'
import RequestDetails from './components/request-details.vue'
import RequestBody from './components/request-body.vue'
import {NewSessionSettings} from './types'
import {
  deleteAllSessionRequests,
  deleteSession,
  deleteSessionRequest,
  getAllSessionRequests,
  getAppSettings,
  getAppVersion,
  getSessionRequest,
  RecordedRequest,
  startNewSession
} from '../api/api'
import ReconnectingWebSocket from 'reconnecting-websocket'
import {newRenewableSessionConnection} from '../websocket/websocket'
import iziToast from 'izitoast'
import routes from './mixins/routes'
import local from './mixins/local'
import {RouteLocationNormalized} from 'vue-router'
import {isValidUUID} from '../utils'

const errorsHandler = console.error

export default defineComponent({
  components: {
    'main-header': MainHeader,
    'request-plate': RequestPlate,
    'requests-navigator': RequestsNavigator,
    'request-details': RequestDetails,
    'index-empty': IndexEmpty,
    'request-body': RequestBody,
  },

  mixins: [
    routes,
    local,
  ],

  data(): {
    requests: RecordedRequest[]

    sessionUUID: string | undefined
    requestUUID: string | undefined

    autoRequestNavigate: boolean
    showRequestDetails: boolean

    maxRequests: number
    sessionLifetimeSec: number
    maxBodySize: number // in bytes
    appVersion: string

    ws: ReconnectingWebSocket | undefined
  } {
    return {
      requests: [] as RecordedRequest[],

      sessionUUID: undefined as string | undefined,
      requestUUID: undefined as string | undefined,

      autoRequestNavigate: true,
      showRequestDetails: true,

      maxRequests: Infinity as number,
      sessionLifetimeSec: Infinity as number,
      maxBodySize: 0 as number, // in bytes
      appVersion: '0.0.0' as string,

      ws: undefined,
    }
  },

  created(): void {
    this.autoRequestNavigate = this.getLocalBool('auto-navigate', true)
    this.showRequestDetails = this.getLocalBool('show-details', true)

    getAppVersion()
      .then((ver) => this.appVersion = ver)
      .catch(errorsHandler)

    getAppSettings()
      .then((s): void => {
        this.maxRequests = s.limits.maxRequests
        this.sessionLifetimeSec = s.limits.sessionLifetimeSec
        this.maxBodySize = s.limits.maxWebhookBodySize
      })
      .catch(errorsHandler)
  },

  computed: {
    currentWebHookUrl: function (): string {
      const uuid = this.sessionUUID
        ? this.sessionUUID
        : '________-____-____-____-____________'

      return `${window.location.origin}${window.location.pathname}${uuid}`
    },
  },

  watch: {
    $route(to: RouteLocationNormalized): void {
      switch (to.name as 'index' | 'request' | undefined) {
        case 'index': {// the index page requested
          const localSessionUUID = this.getLocalSessionUUID()

          if (localSessionUUID) {
            this.navigateToSession(this.$router, localSessionUUID) // redirect to the existing session
          } else {
            this.startNewSession({}) // start a new session with defaults
          }

          break
        }

        case 'request': { // session (+request) page requested
          const {sessionUUID, requestUUID} = to.params as { [key: string]: string | undefined }

          if (typeof sessionUUID !== 'string' || !isValidUUID(sessionUUID)) {
            iziToast.error({title: 'Was requested wrong session ID'})

            this.navigateToIndex(this.$router)
          } else { // valid session UUID requested
            if (sessionUUID !== this.sessionUUID) { // another session was requested
              this.requestUUID = undefined

              getAllSessionRequests(sessionUUID) // reload requests
                .then((requests): void => {
                  this.sessionUUID = sessionUUID

                  this.requests.splice(0, this.requests.length) // make clear
                  this.requests.push(...requests)
                })
                .then((): void => this.renewWebsocketConnection(sessionUUID))
                .then((): void => {
                  if (requestUUID && isValidUUID(requestUUID) && this.requestExist(requestUUID)) {
                    this.switchToRequest(requestUUID) // switch to the requested
                  } else if (!this.requestUUID && this.requests.length) {
                    this.switchToRequest(this.requests[0].UUID) // switch to the first request (if possible)
                  }
                })
                .catch((err): void => {
                  const status: number | undefined = err['status']

                  if (status === 404) { // session was not found
                    this.removeLocalSessionUUID()
                    this.sessionUUID = undefined

                    iziToast.error({title: 'The requested session was not found (or she was expired)'})

                    this.$router.push({name: 'index'}) // redirect to the index page to create a new one
                  } else {
                    errorsHandler(err)
                  }
                })
            }
          }
        }
      }
    },

    requests: { // limit maximal requests length
      deep: true,
      handler(): void {
        if (this.requests.length > this.maxRequests) {
          this.requests.splice(this.maxRequests, this.requests.length - this.maxRequests)

          if (this.requestUUID && !this.requestExist(this.requestUUID)) {
            this.switchToRequest(undefined)
          }
        }
      },
    },

    autoRequestNavigate(v: boolean): void {
      this.setLocalBool('auto-navigate', v)
    },

    showRequestDetails(v: boolean): void {
      this.setLocalBool('show-details', v)
    },
  },

  methods: {
    request(): RecordedRequest | undefined {
      if (this.requestUUID && this.requests.length) {
        for (let i = 0; i < this.requests.length; i++) {
          if (this.requests[i].UUID === this.requestUUID) {
            return this.requests[i]
          }
        }
      }

      return undefined
    },

    requestExist(uuid: string): boolean {
      for (let i = 0; i < this.requests.length; i++) {
        if (this.requests[i].UUID === uuid) {
          return true
        }
      }

      return false
    },

    switchToRequest(uuid: string | undefined): void {
      this.requestUUID = uuid

      if (this.sessionUUID) {
        if (uuid === undefined) {
          this.navigateToSession(this.$router, this.sessionUUID)
        } else {
          this.navigateToRequest(this.$router, this.sessionUUID, uuid)
        }
      }
    },

    renewWebsocketConnection(sessionUUID: string): void {
      if (this.ws) {
        this.ws.close()
        this.ws = undefined
      }

      this.ws = newRenewableSessionConnection(sessionUUID, {
        onRequestRegistered: (requestUUID) => {
          iziToast.info({title: 'New request', message: 'New incoming webhook request', timeout: 2000})

          if (this.sessionUUID) {
            getSessionRequest(this.sessionUUID, requestUUID)
              .then((request) => {
                this.requests.unshift(request) // push at the first position

                if (!this.requestUUID || this.autoRequestNavigate) {
                  this.switchToRequest(requestUUID)
                }
              })
              .catch((err): void => {
                iziToast.error({title: `Cannot load request with UUID ${requestUUID}: ${err.message}`})

                errorsHandler(err)
              })
          }
        },
        onRequestDeleted: (requestUUID) => this.deleteRequest(requestUUID, false),
        onRequestsDeleted: () => this.deleteAllRequests(false),
      })
    },

    startNewSession(urlSettings: NewSessionSettings): void {
      startNewSession({
        contentType: urlSettings.contentType,
        statusCode: urlSettings.statusCode,
        responseDelay: urlSettings.responseDelay,
        responseContent: urlSettings.responseContent,
      })
        .then((sessionData): void => {
          if (urlSettings.destroyCurrentSession === true && this.sessionUUID) {
            deleteSession(this.sessionUUID)
              .catch((err): void => {
                iziToast.error({title: `Cannot destroy current session: ${err.message}`})

                errorsHandler(err)
              })
          }

          this.sessionUUID = sessionData.UUID
          this.requestUUID = undefined

          this.setLocalSessionUUID(sessionData.UUID)
          this.renewWebsocketConnection(sessionData.UUID)
          this.navigateToSession(this.$router, sessionData.UUID)
        })
        .then((): void => this.deleteAllRequests(false))
        .then((): void => iziToast.success({title: 'New session started!'}))
        .catch((err): void => {
          iziToast.error({title: `Cannot create new session: ${err.message}`})

          errorsHandler(err)
        })
    },

    deleteAllRequests(onServer: boolean): void {
      if (onServer && this.sessionUUID) {
        deleteAllSessionRequests(this.sessionUUID)
          .then((success) => {
            if (!success) {
              throw new Error(`All requests removing: unsuccessful status received`)
            }
          })
          .catch((err): void => {
            iziToast.error({title: `Cannot remove all requests: ${err.message}`})

            errorsHandler(err)
          })
      }

      this.switchToRequest(undefined)
      this.requests.splice(0, this.requests.length)
    },

    deleteRequest(requestUUID: string, onServer: boolean): void {
      if (onServer && this.sessionUUID) {
        deleteSessionRequest(this.sessionUUID, requestUUID)
          .then((success) => {
            if (!success) {
              throw new Error(`Request removing: unsuccessful status received`)
            }
          })
          .catch((err): void => {
            iziToast.error({title: `Cannot remove request: ${err.message}`})

            errorsHandler(err)
          })
      }

      let currentRequestIdx: number | undefined = undefined

      for (let i = 0; i < this.requests.length; i++) {
        if (this.requests[i].UUID === requestUUID) {
          currentRequestIdx = i

          break
        }
      }

      if (currentRequestIdx !== undefined) {
        if (requestUUID !== this.requestUUID) {
          // do nothing
        } else if (this.requests[currentRequestIdx + 1]) {
          this.switchToRequest(this.requests[currentRequestIdx + 1].UUID) // navigate to next request
        } else if (this.requests[currentRequestIdx - 1]) {
          this.switchToRequest(this.requests[currentRequestIdx - 1].UUID) // navigate to previous request
        }

        this.requests.splice(currentRequestIdx, 1) // remove request object from stack

        if (this.requests.length === 0) {
          this.switchToRequest(undefined)
        }
      }
    },
  },
})
</script>

<style lang="scss">
$web-font-path: false; // disable external font named "Lato"

@import "~bootswatch/dist/darkly/variables";
@import "~bootstrap/scss/bootstrap";
@import "~bootswatch/dist/darkly/bootswatch";
@import "~izitoast/dist/css/iziToast";
@import "~highlight.js/styles/obsidian.css";

.btn:focus,
.btn:active {
  outline: none !important;
  box-shadow: none;
}

.sidebar {
  flex: 0 0 300px;
}

.hljs {
  background-color: transparent !important;
}

@media (max-width: 690px) {
  .sidebar {
    flex: 0 0 100%;
    width: 100%;
  }
}

.total-requests-count {
  position: relative;
  top: -.15em;
}

.button-delete-all {
  top: -2px;
}
</style>
