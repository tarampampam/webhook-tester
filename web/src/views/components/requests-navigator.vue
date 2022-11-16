<template>
  <div>
    <div
      class="btn-group pb-1"
      role="group"
    >
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        @click="navigateFirstRequest"
        :class="{disabled: requests.length <= 1 || isFirstRequest}"
      >
        <font-awesome-icon
          icon="fa-solid fa-angles-left"
          class="pe-1"
        />
        Newest
      </button>
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        @click="navigatePreviousRequest"
        :class="{disabled: requests.length <= 1 || !requestUUID || isFirstRequest}"
      >
        <font-awesome-icon
          icon="fa-solid fa-angle-left"
          class="pe-1"
        />
        Forward
      </button>
    </div>
    <div
      class="btn-group pb-1 ms-1"
      role="group"
    >
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        @click="navigateNextRequest"
        :class="{disabled: requests.length <= 1 || !requestUUID || isLastRequest}"
      >
        Back
        <font-awesome-icon
          icon="fa-solid fa-angle-right"
          class="ps-1"
        />
      </button>
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        @click="navigateLastRequest"
        :class="{disabled: requests.length <= 1 || isLastRequest}"
      >
        Oldest
        <font-awesome-icon
          icon="fa-solid fa-angles-right"
          class="ps-1"
        />
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent} from 'vue'
import {FontAwesomeIcon} from '@fortawesome/vue-fontawesome'
import {RecordedRequest} from '../../api/api'

export default defineComponent({
  components: {
    'font-awesome-icon': FontAwesomeIcon,
  },

  props: {
    requests: {
      type: Array as () => RecordedRequest[],
      default: () => [],
    },
    requestUUID: {
      type: String,
      default: undefined,
    },
  },

  mounted() {
    window.addEventListener('keydown', this.arrowKeysHandler)
  },

  unmounted() {
    window.removeEventListener('keydown', this.arrowKeysHandler)
  },

  computed: {
    isFirstRequest(): boolean {
      return this.requests.length > 0 && this.requests[0].UUID === this.requestUUID
    },

    isLastRequest(): boolean {
      return this.requests.length > 0 && this.requests[this.requests.length - 1].UUID === this.requestUUID
    },
  },

  methods: {
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

    arrowKeysHandler(e: KeyboardEvent): void {
      if (e.code === 'ArrowDown' || e.code === 'ArrowRight') {
        this.navigateNextRequest()
      } else if (e.code === 'ArrowUp' || e.code === 'ArrowLeft') {
        this.navigatePreviousRequest()
      }
    },

    navigateFirstRequest(): void {
      const first = this.requests[0]

      if (first && first.UUID !== this.requestUUID) {
        this.$emit('changed', first.UUID)
      }
    },
    navigatePreviousRequest(): void {
      const current = this.getCurrentRequestIndex()

      if (current !== undefined) {
        const prev = this.requests[current - 1]

        if (prev && prev.UUID !== this.requestUUID) {
          this.$emit('changed', prev.UUID)
        }
      }
    },
    navigateNextRequest(): void {
      const current = this.getCurrentRequestIndex()

      if (current !== undefined) {
        const next = this.requests[current + 1]

        if (next && next.UUID !== this.requestUUID) {
          this.$emit('changed', next.UUID)
        }
      }
    },
    navigateLastRequest(): void {
      const last = this.requests[this.requests.length - 1]

      if (last && last.UUID !== this.requestUUID) {
        this.$emit('changed', last.UUID)
      }
    },
  },

  emits: [
    'changed'
  ],
})
</script>

<style lang="scss" scoped>
</style>
