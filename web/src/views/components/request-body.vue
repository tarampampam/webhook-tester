<template>
  <div>
    <ul class="nav nav-pills">
      <li class="nav-item">
        <span
          class="btn nav-link ps-4 pe-4 pt-1 pb-1"
          :class="{ 'active': mode === 'text' }"
          @click="mode='text'"
        >
          <font-awesome-icon icon="fa-solid fa-font" /> Text view
        </span>
      </li>
      <li class="nav-item">
        <span
          class="btn nav-link pl-4 pr-4 pt-1 pb-1"
          :class="{ 'active': mode === 'binary' }"
          @click="mode='binary'"
        >
          <font-awesome-icon icon="fa-solid fa-atom" /> Binary view
        </span>
      </li>
      <li
        class="nav-item"
        v-if="request && request.content.length"
      >
        <span
          class="btn nav-link pl-4 pr-4 pt-1 pb-1"
          @click="download"
        >
          <font-awesome-icon icon="fa-solid fa-download" /> Download
        </span>
      </li>
    </ul>
    <div class="tab-content pt-2 pb-2">
      <div
        class="alert alert-secondary"
        v-if="request?.content.byteLength > 2048 && !displayLargeContent"
      >
        <p>
          The request body is large and is not displayed by default due to performance reasons.
        </p>
        <p class="mb-0">
          <span
            class="btn btn-primary btn-sm px-4"
            @click="displayLargeContent = true"
          >Click here to display it</span>
        </p>
      </div>
      <div
        class="tab-pane active"
        v-else-if="mode === 'text'"
      >
        <highlightjs
          autodetect
          class="highlightjs"
          :code="content(true)"
        />
      </div>
      <div
        class="tab-pane active pt-2"
        v-else-if="mode === 'binary'"
      >
        <hex-view :content="request?.content" />
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent} from 'vue'
import {FontAwesomeIcon} from '@fortawesome/vue-fontawesome'
import {RecordedRequest} from '../../api/api'
import hljsVuePlugin from '@highlightjs/vue-plugin'
import HexView from './hex-view.vue'

const textDecoder = new TextDecoder('utf-8')

export default defineComponent({
  components: {
    'font-awesome-icon': FontAwesomeIcon,
    'highlightjs': hljsVuePlugin.component,
    'hex-view': HexView,
  },

  props: {
    request: {
      type: Object as () => RecordedRequest | undefined,
      default: undefined,
    },
  },

  data(): {
    mode: 'text' | 'binary'
    displayLargeContent: boolean
  } {
    return {
      mode: 'text',
      displayLargeContent: false,
    }
  },

  methods: {
    content(pretty: boolean): string {
      if (this.request && this.request.content.length) {
        const asString = textDecoder.decode(this.request.content)

        if (pretty) {
          try { // decorate json
            return JSON.stringify(JSON.parse(asString), undefined, 2)
          } catch (e) {
            // wrong json
          }
        }

        return asString
      }

      return ''
    },

    download(): void {
      if (this.request && this.request.content.length) {
        const $body = document.body
        const $a = document.createElement('a')
        const raw = encodeURIComponent(textDecoder.decode(this.request.content))

        $a.setAttribute('href', 'data:application/octet-stream;charset=utf-8,' + raw)
        $a.setAttribute('download', this.request.UUID + '.bin')
        $a.style.display = 'none'

        $body.appendChild($a)
        $a.click()
        $body.removeChild($a)
      }
    }
  },
  watch: {
    request(): void {
      this.displayLargeContent = false
    }
  },
})
</script>

<style lang="scss" scoped>
.highlightjs {
  tab-size: 2;
  margin-bottom: 0;
  word-wrap: break-word;
  white-space: pre-wrap;
}
</style>
