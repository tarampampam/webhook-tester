<template>
  <div class="request-plate list-group-item list-group-item-action flex-column py-3 px-3" :class="methodClass">
    <div class="d-flex w-100 justify-content-between">
      <h5 class="mb-1 text-nowrap">{{ clientAddress }}</h5>
      <button type="button" class="btn-close position-relative small" title="Delete" @click="remove"></button>
    </div>
    <p class="when small m-0">{{ formattedWhen }}</p>
  </div>
</template>

<script lang="ts">
import {defineComponent} from 'vue'
import moment from 'moment'

export default defineComponent({
  props: {
    uuid: {
      type: String,
    },
    clientAddress: {
      type: String,
      default: 'XXX.XXX.XXX.XXX',
    },
    method: {
      type: String,
      default: '',
    },
    when: {
      type: Date,
      default: undefined,
    },
  },

  data() {
    return {
      intervalId: undefined as undefined | number,
      formattedWhen: '',
    }
  },

  mounted(): void {
    this.updateFormattedWhen()

    this.intervalId = window.setInterval(() => this.updateFormattedWhen(), 150)
  },

  beforeDestroy(): void {
    window.clearInterval(this.intervalId)
  },

  computed: {
    methodClass: function (): string {
      if (typeof this.method === 'string') {
        switch (this.method.toLowerCase()) {
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
      this.$emit('onDelete', this.uuid)

      e.preventDefault()
      e.stopPropagation()

      this.updateFormattedWhen()
    },

    updateFormattedWhen(): void {
      this.formattedWhen = this.when
        ? `${moment(this.when).format('h:mm:ss a')} (${moment(this.when).fromNow()})`
        : ''
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
