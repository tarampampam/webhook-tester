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
      <div class="sidebar col-sm-5 col-md-4 col-lg-3 col-xl-2 px-2 py-0">
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
            :uuid="r.UUID"
            :client-address="r.clientAddress"
            :method="r.method"
            :when="r.createdAt"
            :class="{ active: requestUUID === r.UUID }"
            @click.native="requestUUID = r.UUID"
            @onDelete="deleteRequestHandler"
          ></request-plate>
        </div>
        <div v-else class="text-muted text-center mt-3">
          <span class="spinner-border spinner-border-sm me-1"></span> Waiting for first request
        </div>
      </div>

      <div class="col-sm-7 col-md-8 col-lg-9 col-xl-10 py-3 ps-md-4" role="main">
        <div v-if="requests.length > 0 && requestUUID">
          <div class="row pt-2">
            <div class="col-6">
              <div class="btn-group pb-1" role="group">
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  @click.prevent="navigateFirstRequest"
                  :class="{disabled: requests.length <= 1}"
                >
                  <font-awesome-icon icon="fa-solid fa-angles-left" class="pe-1"/>
                  First request
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  @click="navigatePreviousRequest"
                  :class="{disabled: requests.length <= 1 || !requestUUID}"
                >
                  <font-awesome-icon icon="fa-solid fa-angle-left" class="pe-1"/>
                  Previous
                </button>
              </div>
              <div class="btn-group pb-1 ms-1" role="group">
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  @click="navigateNextRequest"
                  :class="{disabled: requests.length <= 1 || !requestUUID}"
                >
                  Next
                  <font-awesome-icon icon="fa-solid fa-angle-right" class="ps-1"/>
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  @click="navigateLastRequest"
                  :class="{disabled: requests.length <= 1}"
                >
                  Last request
                  <font-awesome-icon icon="fa-solid fa-angles-right" class="ps-1"/>
                </button>
              </div>
            </div>
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
            :request="getRequestByUUID(requestUUID)"
            :uuid="requestUUID"
          ></request-details>

          <div class="pt-3">
            <h4>Request body</h4>

            <div v-if="requestContentExists">
              <ul class="nav nav-pills">
                <li class="nav-item">
                  <span
                    class="btn nav-link ps-4 pe-4 pt-1 pb-1"
                    :class="{ 'active': requestContentViewMode === 'text' }"
                    @click="requestContentViewMode='text'"
                  >
                    <font-awesome-icon icon="fa-solid fa-font"/> Text view
                  </span>
                </li>
                <li class="nav-item">
                  <span
                    class="btn nav-link pl-4 pr-4 pt-1 pb-1"
                    :class="{ 'active': requestContentViewMode === 'binary' }"
                    @click="requestContentViewMode='binary'"
                  >
                    <font-awesome-icon icon="fa-solid fa-atom"/> Binary view
                  </span>
                </li>
                <li
                  class="nav-item"
                  v-if="getRequestByUUID(requestUUID)"
                >
                  <span
                    class="btn nav-link pl-4 pr-4 pt-1 pb-1"
                    @click="handleDownloadRequestContent"
                  >
                    <font-awesome-icon icon="fa-solid fa-download"/> Download
                  </span>
                </li>
              </ul>
              <div class="tab-content pt-2 pb-2">
                <div
                  class="tab-pane active"
                  v-if="requestContentViewMode === 'text'"
                >
                  <highlightjs autodetect :code="requestContentPretty" />
                </div>
                <div
                  class="tab-pane active pt-2"
                  v-if="requestContentViewMode === 'binary'"
                >
                <hex-view :content="requestBinaryContent"></hex-view>
                </div>
              </div>
            </div>
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
import RequestDetails from './components/request-details.vue'
import HexView from './components/hex-view.vue'
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
import hljsVuePlugin from "@highlightjs/vue-plugin";

const textDecoder = new TextDecoder('utf-8')

const errorsHandler = console.error

export default defineComponent({
  components: {
    'font-awesome-icon': FontAwesomeIcon,
    'main-header': MainHeader,
    'request-plate': RequestPlate,
    'request-details': RequestDetails,
    'index-empty': IndexEmpty,
    'hex-view': HexView,
    'highlightjs': hljsVuePlugin.component,
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
  mounted() {
    document.getElementById('main-loader')?.remove() // hide main loading spinner
  },
  computed: {
    sessionRequestURI: function (): string {
      const uuid = this.sessionUUID
        ? this.sessionUUID
        : '________-____-____-____-____________'

      return `${window.location.origin}/${uuid}`
    },

    requestContentExists: function (): boolean {
      if (this.requestUUID) {
        const request = this.getRequestByUUID(this.requestUUID)

        return request !== undefined && request.content.length > 0
      }

      return false
    },

    requestContent: function (): string {
      if (this.requestUUID) {
        const request = this.getRequestByUUID(this.requestUUID)

        if (request && request.content.length > 0) {
          return textDecoder.decode(request.content)
        }
      }

      return ''
    },

    requestContentPretty: function (): string {
      try { // decorate json
        return JSON.stringify(JSON.parse(this.requestContent), undefined, 2)
      } catch (e) {
        // wrong json
      }

      return ''
    },

    requestBinaryContent: function (): Uint8Array {
      if (this.requestUUID) {
        const request = this.getRequestByUUID(this.requestUUID)

        if (request && request.content.length > 0) {
          return request.content
        }
      }

      return new Uint8Array(0)
    },
  },

  watch: {
    sessionUUID() {
      if (this.$route.params.sessionUUID !== this.sessionUUID) {
        this.$router.push({
          name: 'request',
          params: {sessionUUID: this.sessionUUID}
        }).catch(errorsHandler)
      }

      this.wsRefreshConnection()
    },
    requestUUID() {
      if (this.$route.params.requestUUID !== this.requestUUID) {
        this.$router.push({
          name: 'request', params: {
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
          if (!this.getRequestByUUID(this.requestUUID)) {
            this.requestUUID = undefined
          }
        }
      }
    },
  },

  methods: {
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
                      this.navigateFirstRequest()
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
    handleDownloadRequestContent(): void {
      if (this.requestUUID) {
        const request = this.getRequestByUUID(this.requestUUID)

        if (request && request.content.length > 0) {
          const $body = document.body
          const $a = document.createElement('a')
          const raw = encodeURIComponent(textDecoder.decode(request.content))

          $a.setAttribute('href', 'data:application/octet-stream;charset=utf-8,' + raw)
          $a.setAttribute('download', this.requestUUID + '.bin')
          $a.style.display = 'none'

          $body.appendChild($a)
          $a.click()
          $body.removeChild($a)
        }
      }
    },

    deleteRequest(requestUUID: string): void {
      const currentIndex = this.getRequestIndexByUUID(requestUUID)

      if (currentIndex !== undefined) {
        if (requestUUID !== this.requestUUID) {
          // do nothing
        } else if (this.requests[currentIndex + 1]) {
          this.navigateNextRequest()
        } else if (this.requests[currentIndex - 1]) {
          this.navigatePreviousRequest()
        }

        this.requests.splice(currentIndex, 1) // remove request object from stack
      }
    },

    clearRequests(): void {
      this.requests.splice(0, this.requests.length)
      this.requestUUID = undefined
    },

    getCurrentRequestIndex(): number | undefined {
      if (this.requests.length > 0) {
        for (let i = 0; i < this.requests.length; i++) {
          if (this.requests[i].UUID === this.requestUUID) {
            return i
          }
        }
      }

      return undefined
    },

    getRequestByUUID(uuid: string): RecordedRequest | undefined {
      if (this.requests.length > 0) {
        for (let i = 0; i < this.requests.length; i++) {
          if (this.requests[i].UUID === uuid) {
            return this.requests[i]
          }
        }
      }

      return undefined
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
            if (!this.requestUUID || this.getRequestIndexByUUID(this.requestUUID) === undefined) {
              this.navigateFirstRequest()
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

    getRequestIndexByUUID(uuid: string): number | undefined {
      for (let i = 0; i < this.requests.length; i++) {
        if (this.requests[i].UUID === uuid) {
          return i
        }
      }

      return undefined
    },

    navigateFirstRequest(): void {
      const first = this.requests[0]

      if (first && first.UUID !== this.requestUUID) {
        this.requestUUID = first.UUID
      }
    },
    navigatePreviousRequest(): void {
      const current = this.getCurrentRequestIndex()

      if (current !== undefined) {
        const prev = this.requests[current - 1]

        if (prev && prev.UUID !== this.requestUUID) {
          this.requestUUID = prev.UUID
        }
      }
    },
    navigateNextRequest(): void {
      const current = this.getCurrentRequestIndex()

      if (current !== undefined) {
        const next = this.requests[current + 1]

        if (next && next.UUID !== this.requestUUID) {
          this.requestUUID = next.UUID
        }
      }
    },
    navigateLastRequest(): void {
      const last = this.requests[this.requests.length - 1]

      if (last && last.UUID !== this.requestUUID) {
        this.requestUUID = last.UUID
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

.total-requests-count {
  position: relative;
  top: -.15em;
}

.request-plate {
  cursor: pointer;
}

.button-delete-all {
  top: -2px;
}

.hljs {
  background-color: transparent;
}
</style>
