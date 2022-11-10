<template>
  <div
    class="request-plate list-group-item list-group-item-action flex-column py-3 px-3"
    :class="methodClass"
  >
    <div class="d-flex w-100 justify-content-between">
      <h5 class="mb-1 text-nowrap">
        {{ request.clientAddress }}
      </h5>
      <button
        type="button"
        class="btn-close position-relative small"
        title="Delete"
        @click="remove"
      />
    </div>
    <p class="when small m-0">
      {{ formattedWhen }}
    </p>
  </div>
</template>

<script lang="ts">
import {defineComponent} from 'vue'
import moment from 'moment'
import {RecordedRequest} from '../../api/api'

export default defineComponent({
  props: {
    request: {
      type: Object as () => RecordedRequest | undefined,
      default: undefined,
    },
  },

  data(): {
    intervalId: undefined | number
    formattedWhen: string
  } {
    return {
      intervalId: undefined as undefined | number,
      formattedWhen: '',
    }
  },

  mounted(): void {
    this.updateFormattedWhen()

    this.intervalId = window.setInterval(this.updateFormattedWhen, 150)
  },

  watch: {
    request(): void {
      this.updateFormattedWhen() // force update
    },
  },

  beforeDestroy(): void {
    window.clearInterval(this.intervalId)
  },

  computed: {
    methodClass: function (): string {
      if (this.request) {
        switch (this.request.method.toLowerCase()) {
          case 'get':
            return 'border-success'
          case 'head':
            return 'border-light'
          case 'post':
            return 'border-primary'
          case 'put':
            return 'border-info'
          case 'patch':
            return 'border-dark'
          case 'delete':
            return 'border-danger'
          case 'connect':
            return 'border-warning'
          case 'options':
            return 'border-secondary'
          case 'trace':
            return 'border-light'
        }
      }

      return ''
    }
  },

  methods: {
    remove(e): void {
      if (this.request) {
        this.$emit('onDelete', this.request.UUID)

        e.preventDefault()
        e.stopPropagation()

        this.updateFormattedWhen()
      }
    },

    updateFormattedWhen(): void {
      if (this.request) {
        this.formattedWhen = `${moment(this.request.createdAt).format('h:mm:ss a')} (${moment(this.request.createdAt).fromNow()})`

        return
      }

      this.formattedWhen = ''
    }
  },
  emits: [
    'onDelete',
  ],
})
</script>

<style lang="scss" scoped>
.request-plate {
  border-width: 0 0 0 0.3em;
  border-top-color: transparent !important;
  border-right-color: transparent !important;
  border-bottom-color: transparent !important;
  cursor: pointer;

  .when {
    line-height: 1em;
    opacity: 0.8;
  }

  .close {
    top: -5px;
  }
}
</style>
