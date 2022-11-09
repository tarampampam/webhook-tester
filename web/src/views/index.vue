<template>
  <main-header
    :currentWebHookUrl="sessionRequestURI"
    :sessionLifetimeSec="sessionLifetimeSec"
    :maxBodySizeBytes="maxBodySize"
    :version="appVersion"
    @createNewUrl="newSessionHandler"
  ></main-header>

  <div class="container-fluid mb-2">
    <div class="row flex-xl-nowrap">
      <div class="sidebar px-2 py-0">
        <div class="ps-3 pt-4 pe-3 pb-3">
          <div class="d-flex w-100 justify-content-between">
            <h5 class="text-uppercase mb-0">
              Requests
              <span class="badge bg-primary rounded-pill ms-1 total-requests-count">{{ requests.length }}</span>
            </h5>
            <button type="button"
                    class="btn btn-outline-danger btn-sm position-relative button-delete-all"
                    v-if="requests.length > 0"
                    @click="deleteAllRequestsHandler">
              Delete all
            </button>
          </div>
        </div>

        <div class="list-group" v-if="requests.length > 0">
          <request-plate
            v-for="r in requests"
            :key="r.UUID"
            :request="r"
            :class="{ active: requestUUID === r.UUID }"
            @click="requestUUID = r.UUID"
            @onDelete="(uuid: string) => deleteRequestHandler(uuid)"
          ></request-plate>
        </div>
        <div v-else class="text-muted text-center mt-3">
          <span class="spinner-border spinner-border-sm me-1"></span> Waiting for first request
        </div>
      </div>

      <div class="col py-3 ps-md-4" role="main">
        <div v-if="requests.length > 0">
          <div class="row pt-2">
            <requests-navigator
              class="col-6"
              :requests="requests"
              :requestUUID="requestUUID"
              @changed="(uuid: string) => requestUUID = uuid"
            />
            <div class="col-6 pb-1 text-end">
              <div class="form-check d-inline-block">
                <input
                  type="checkbox"
                  class="form-check-input"
                  id="show-details"
                  v-model="showRequestDetails"
                >
                <label class="form-check-label" for="show-details">Show details</label>
              </div>
              <div class="form-check d-inline-block ms-3"
                   title="Automatically select and go to the latest incoming webhook request">
                <input
                  type="checkbox"
                  class="form-check-input"
                  id="auto-navigate"
                  v-model="autoRequestNavigate"
                >
                <label class="form-check-label" for="auto-navigate">Auto navigate</label>
              </div>
            </div>
          </div>

          <request-details
            v-if="showRequestDetails"
            class="pt-3"
            :request="request()"
          ></request-details>

          <div class="pt-3">
            <h4>Request body</h4>

            <request-body
              v-if="request() && request().content.length"
              :request="request()"
            />
            <div v-else class="pt-1 pb-1">
              <p class="text-muted small text-monospace">// empty request body</p>
            </div>
          </div>
        </div>
        <index-empty
          v-else
          :currentWebHookUrl="sessionRequestURI"
        ></index-empty>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent} from 'vue'
import IndexEmpty from './components/index-empty.vue'
import MainHeader from './components/main-header.vue'
import RequestPlate from './components/request-plate.vue'
import RequestsNavigator from './components/requests-navigator.vue'
import RequestDetails from './components/request-details.vue'
import RequestBody from './components/request-body.vue'
import {FontAwesomeIcon} from '@fortawesome/vue-fontawesome'
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
import {getLocalSessionUUID, setLocalSessionUUID} from '../session'
import {isValidUUID} from '../utils'

const textDecoder = new TextDecoder('utf-8')

const errorsHandler = console.error

export default defineComponent({
  components: {
    'font-awesome-icon': FontAwesomeIcon,
    'main-header': MainHeader,
    'request-plate': RequestPlate,
    'requests-navigator': RequestsNavigator,
    'request-details': RequestDetails,
    'index-empty': IndexEmpty,
    'request-body': RequestBody,
  },
  data(): {
    requests: RecordedRequest[]

    sessionUUID: string | undefined
    requestUUID: string | undefined

    autoRequestNavigate: boolean
    showRequestDetails: boolean
    requestContentViewMode: 'text' | 'binary'

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
      requestContentViewMode: 'text' as 'text' | 'binary',

      maxRequests: Infinity as number,
      sessionLifetimeSec: Infinity as number,
      maxBodySize: 0 as number, // in bytes
      appVersion: '0.0.0' as string,

      ws: undefined,
    }
  },
  created(): void {
    // this.$router.beforeEach((from, to): boolean => { // false: cancel the current navigation, true: next navigation guard is called
    //   console.log(from, to)
    //
    //   const {sessionUUID, requestUUID} = to.params
    //
    //   if (typeof sessionUUID === 'string') {
    //     this.sessionUUID = sessionUUID
    //   }
    //
    //   if (typeof requestUUID === 'string') {
    //     this.requestUUID = requestUUID
    //   }
    //
    //   return true
    // })

    getAppVersion()
      .then((ver) => this.appVersion = ver)
      .catch(errorsHandler)

    getAppSettings()
      .then((s) => {
        this.maxRequests = s.limits.maxRequests
        this.sessionLifetimeSec = s.limits.sessionLifetimeSec
        this.maxBodySize = s.limits.maxWebhookBodySize
      })
      .catch(errorsHandler)

    this.wsRefreshConnection()

    this.initSession()
    this.initRequest()
  },
  computed: {
    sessionRequestURI: function (): string {
      const uuid = this.sessionUUID
        ? this.sessionUUID
        : '________-____-____-____-____________'

      return `${window.location.origin}/${uuid}`
    },
  },

  watch: {
    sessionUUID() {
      if (this.$route.params.sessionUUID !== this.sessionUUID) {
        this.$router.push({
          name: 'request',
          params: {
            sessionUUID: this.sessionUUID,
          }
        }).catch(errorsHandler)
      }

      this.wsRefreshConnection()
    },
    requestUUID() {
      if (this.$route.params.requestUUID !== this.requestUUID) {
        this.$router.push({
          name: 'request',
          params: {
            sessionUUID: this.sessionUUID,
            requestUUID: this.requestUUID,
          }
        }).catch(errorsHandler)
      }
    },
    requests() {
      // limit maximal requests length
      if (this.requests.length > this.maxRequests) {
        this.requests.splice(this.maxRequests, this.requests.length)

        if (this.requestUUID) {
          if (!this.request()) {
            this.requestUUID = undefined
          }
        }
      }
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

    wsRefreshConnection(): void {
      enum names {
        requestRegistered = 'request-registered',
        requestDeleted = 'request-deleted',
        requestsDeleted = 'requests-deleted',
      }

      if (this.ws) {
        this.ws.close()
        this.ws = undefined
      }

      if (this.sessionUUID) {
        this.ws = newRenewableSessionConnection(this.sessionUUID, (name, data): void => {
          switch (name) { // route incoming events
            case names.requestRegistered: {
              const requestUUID = data

              iziToast.info({
                title: 'New request',
                message: 'New incoming webhook request',
                timeout: 2000,
                closeOnClick: true,
              })

              if (this.sessionUUID) {
                getSessionRequest(this.sessionUUID, requestUUID)
                  .then((request) => {
                    this.requests.unshift(request) // push at the first position

                    if (!this.requestUUID || this.autoRequestNavigate) {
                      this.requestUUID = requestUUID
                    }
                  })
                  .catch((err): void => {
                    iziToast.error({title: `Cannot load request with UUID ${requestUUID}: ${err.message}`})

                    errorsHandler(err)
                  })
              }

              break
            }

            case names.requestDeleted: {
              this.deleteRequest(data)

              break
            }

            case names.requestsDeleted: {
              this.clearRequests()

              break
            }
          }
        })
      }
    },

    newSessionHandler(urlSettings: NewSessionSettings): void {
      startNewSession({
        contentType: urlSettings.contentType,
        statusCode: urlSettings.statusCode,
        responseDelay: urlSettings.responseDelay,
        responseContent: urlSettings.responseContent,
      })
        .then((newSessionData) => {
          if (urlSettings.destroyCurrentSession === true && this.sessionUUID) {
            deleteSession(this.sessionUUID)
              .catch((err): void => {
                iziToast.error({title: `Cannot destroy current session: ${err.message}`})

                errorsHandler(err)
              })
          }

          this.sessionUUID = newSessionData.UUID
          setLocalSessionUUID(newSessionData.UUID)

          this.clearRequests()
          iziToast.success({title: 'New session started!'})
        })
        .catch((err): void => {
          iziToast.error({title: `Cannot create new session: ${err.message}`})

          errorsHandler(err)
        })
    },
    deleteAllRequestsHandler(): void {
      if (this.sessionUUID) {
        deleteAllSessionRequests(this.sessionUUID)
          .then((success) => {
            if (success) {
              iziToast.success({title: 'All requests successfully removed!'})
            } else {
              throw new Error(`I've got unsuccessful status`)
            }
          })
          .catch((err): void => {
            iziToast.error({title: `Cannot remove all requests: ${err.message}`})

            errorsHandler(err)
          })

        this.clearRequests()
      }
    },

    deleteRequestHandler(requestUUID: string): void {
      if (this.sessionUUID) {
        deleteSessionRequest(this.sessionUUID, requestUUID)
          .then((success) => {
            if (!success) {
              throw new Error(`Unsuccessful status returned`)
            }
          })
          .catch((err): void => {
            iziToast.error({title: `Cannot remove request: ${err.message}`})

            errorsHandler(err)
          })
      }

      this.deleteRequest(requestUUID)
    },

    deleteRequest(requestUUID: string): void {
      if (this.requestUUID) {
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
            this.requestUUID = this.requests[currentRequestIdx + 1].UUID // navigate to next request
          } else if (this.requests[currentRequestIdx - 1]) {
            this.requestUUID = this.requests[currentRequestIdx - 1].UUID // navigate to previous request
          }

          this.requests.splice(currentRequestIdx, 1) // remove request object from stack
        }
      }
    },

    clearRequests(): void {
      this.requests.splice(0, this.requests.length)
      this.requestUUID = undefined
    },

    initSession(): void {
      const localSessionUUID = getLocalSessionUUID()
      const routeSessionUUID = this.$route.params.sessionUUID as string | undefined

      const reloadRequests = (): Promise<void> => {
        return new Promise((resolve, reject) => {
          if (this.sessionUUID) {
            getAllSessionRequests(this.sessionUUID)
              .then((requests) => {
                this.requests.splice(0, this.requests.length) // make clear
                requests.forEach((request) => this.requests.push(request))

                resolve()
              })
              .catch(reject)
          } else {
            resolve()
          }
        })
      }

      const newSession = (): void => {
        startNewSession({})
          .then((newSessionData) => {
            this.sessionUUID = newSessionData.UUID
            setLocalSessionUUID(newSessionData.UUID)

            reloadRequests()
              .catch((err) => iziToast.error({title: `Cannot retrieve requests: ${err.message}`}))
          })
          .catch((err) => iziToast.error({title: `Cannot create new session: ${err.message}`}))
      }

      const sessionUUID = isValidUUID(routeSessionUUID)
        ? routeSessionUUID
        : (isValidUUID(localSessionUUID) ? localSessionUUID : undefined)

      if (sessionUUID) {
        this.sessionUUID = sessionUUID

        reloadRequests()
          .then(() => {
            if (!this.requestUUID || !this.request()) {
              if (this.requests[0]) {
                this.requestUUID = this.requests[0].UUID // navigate first request
              }
            }
          })
          .catch((): void => newSession())
      } else {
        newSession()
      }
    },

    initRequest(): void {
      const routeRequestUUID = this.$route.params.requestUUID as string | undefined

      if (isValidUUID(routeRequestUUID)) {
        this.requestUUID = routeRequestUUID
      }
    },
  },
})
</script>

<style lang="scss">
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
