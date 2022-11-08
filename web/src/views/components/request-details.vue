<template>
  <div class="row request-details">
    <div class="col-md-12 col-lg-5 col-xl-4">
      <div class="row">
        <div class="col-7">
          <h4>Request details</h4>
        </div>
        <div class="col-5 text-end">
          <button
            class="btn btn-primary btn-sm"
            v-bind:data-clipboard-text="permalink"
            type="button"
            data-clipboard
          >
            Copy permalink
          </button>
        </div>
      </div>

      <div class="row pb-1">
        <div class="col-lg-3 text-lg-end">URL</div>
        <div class="col-lg-9 text-break">
          <code><a :href="getRequestURI">{{ getRequestURI }}</a></code>
        </div>
      </div>

      <div class="row pb-1">
        <div class="col-lg-3 text-lg-end">Method</div>
        <div class="col-lg-9">
          <span class="badge text-uppercase" :class="methodClass">{{ request.method.toUpperCase() }}</span>
        </div>
      </div>

      <div class="row pb-1">
        <div class="col-lg-3 text-lg-end">From</div>
        <div class="col-lg-9">
          <a :href="'https://who.is/whois-ip/ip-address/' + request.clientAddress"
             target="_blank"
             rel="noreferrer"
             title="WhoIs?"
          >
            <strong>{{ request.clientAddress }}</strong>
          </a>
        </div>
      </div>

      <div class="row pb-1">
        <div class="col-lg-3 text-lg-end">When</div>
        <div class="col-lg-9">
          <span>{{ formattedWhen }}</span>
        </div>
      </div>

      <div class="row pb-1">
        <div class="col-lg-3 text-lg-end">Size</div>
        <div class="col-lg-9">
          <span v-if="contentLength">{{ contentLength }} bytes</span>
          <span v-else class="text-muted">&mdash;</span>
        </div>
      </div>

      <div class="row pb-1">
        <div class="col-lg-3 text-lg-end">ID</div>
        <div class="col-lg-9 text-break">
          <code>{{ uuid }}</code>
        </div>
      </div>
    </div>

    <div class="col-md-12 col-lg-7 col-xl-8 mt-3 mt-md-3 mt-lg-0" v-if="this.request.headers">
      <h4 class="row">
        <div class="col-lg-4 col-xl-3"></div>
        <div class="col-lg-8 col-xl-9">
          HTTP headers
        </div>
      </h4>
      <div v-for="(header) in this.request.headers"
           :key="header.name"
           class="row pb-1">
        <div class="col-lg-4 col-xl-3 text-lg-end">
          {{ header.name }}
        </div>
        <div class="col-lg-8 col-xl-9 text-break">
          <code>{{ header.value }}</code>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent} from 'vue'
import moment from 'moment'
import {RecordedRequest} from '../../api/api'

export default defineComponent({
  props: {
    request: {
      type: Object as () => RecordedRequest,
      default: null,
    },
    uuid: {
      type: String,
      default: null,
    },
  },

  data(): {
    intervalId: number | undefined
    formattedWhen: string
    permalink: string
  } {
    return {
      intervalId: undefined,
      formattedWhen: '',
      permalink: window.location.href,
    }
  },

  watch: {
    uuid(): void {
      this.updateFormattedWhen()
      this.permalink = window.location.href // force update
    },
  },

  mounted(): void {
    this.updateFormattedWhen()
    this.intervalId = window.setInterval(() => this.updateFormattedWhen(), 1000)
  },

  computed: {
    getRequestURI: function (): string {
      let uri = (typeof this.request === 'object' && this.request && typeof this.request.url === 'string')
        ? this.request.url.replace(/^\/+/g, '')
        : '...'

      return `${window.location.origin}/${uri}`
    },

    methodClass: function (): string {
      if (this.request && this.request.method) {
        switch (this.request.method.toLowerCase()) {
          case 'get':
            return 'text-bg-success'
          case 'post':
          case 'put':
            return 'text-bg-info'
          case 'delete':
            return 'text-bg-danger'
        }
      }

      return 'text-bg-light'
    },

    contentLength(): number {
      return this.request ? this.request.content.length : 0
    },
  },

  beforeDestroy: function (): void {
    window.clearInterval(this.intervalId)
  },

  methods: {
    updateFormattedWhen(): void {
      this.formattedWhen = (this.request && this.request.createdAt)
        ? `${moment(this.request.createdAt).format('YYYY-MM-D h:mm:ss a')} (${moment(this.request.createdAt).fromNow()})`
        : ''
    }
  }
})
</script>

<style lang="scss" scoped>
.request-details .text-break {
  word-break: break-all;
}
</style>
