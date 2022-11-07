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
            v-for="r in this.requests"
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
        <div v-if="requests.length > 0 && !this.requestUUID">
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
                  :class="{disabled: requests.length <= 1 || !this.requestUUID}"
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
                  :class="{disabled: requests.length <= 1 || !this.requestUUID}"
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
            :request="getRequestByUUID(this.requestUUID)"
            :uuid="this.requestUUID"
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
                  v-if="getRequestByUUID(this.requestUUID)"
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
                  <!--                  <pre v-highlightjs="requestContent"><code></code></pre>-->
                </div>
                <div
                  class="tab-pane active pt-2"
                  v-if="requestContentViewMode === 'binary'"
                >
                  <!--                  <hex-view-->
                  <!--                    :content="requestBinaryContent"-->
                  <!--                  ></hex-view>-->
                </div>
              </div>
            </div>
            <div v-else class="pt-1 pb-1">
              <p class="text-muted small text-monospace">// empty request body</p>
            </div>
          </div>
        </div>
        <!--        <index-empty-->
        <!--          v-else-->
        <!--          :current-web-hook-url="sessionRequestURI"-->
        <!--        ></index-empty>-->
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent} from 'vue'
import MainHeader from './components/main-header.vue'
import RequestPlate from './components/request-plate.vue'
import RequestDetails from './components/request-details.vue'
import {FontAwesomeIcon} from '@fortawesome/vue-fontawesome'
import {NewSessionSettings, RecordedRequest} from './types'

export default defineComponent({
  components: {
    'font-awesome-icon': FontAwesomeIcon,
    'main-header': MainHeader,
    'request-plate': RequestPlate,
    'request-details': RequestDetails,
  },
  data() {
    return {
      requests: [{} as RecordedRequest, {} as RecordedRequest] as RecordedRequest[],

      sessionUUID: undefined as string | undefined,
      requestUUID: undefined as string | undefined,

      autoRequestNavigate: true,
      showRequestDetails: true,
      requestContentViewMode: 'text' as 'text' | 'binary',

      sessionLifetimeSec: Infinity as number,
      maxBodySize: 0 as number, // in bytes
      appVersion: '0.0.0' as string,
    }
  },
  created(): void {
    //
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
  },
  methods: {
    newSessionHandler(urlSettings: NewSessionSettings): void {
      console.log(urlSettings)
    },
    deleteAllRequestsHandler(): void {
      console.log('deleteAllRequestsHandler')
    },
    deleteRequestHandler(): void {
      console.log('deleteRequestHandler')
    },
    handleDownloadRequestContent(): void {
      console.log('handleDownloadRequestContent')
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

    navigateFirstRequest(): void {
      const first = this.requests[0]

      if (first && first.UUID !== this.requestUUID) {
        this.requestUUID = first.UUID
      }
    },
    navigatePreviousRequest(): void {
      const current = this.getCurrentRequestIndex()

      if (current) {
        const prev = this.requests[current - 1]

        if (prev && prev.UUID !== this.requestUUID) {
          this.requestUUID = prev.UUID
        }
      }
    },
    navigateNextRequest(): void {
      const current = this.getCurrentRequestIndex()

      if (current) {
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
</style>
