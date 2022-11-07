<template>
  <main-header
    :currentWebHookUrl="sessionRequestURI"
    :sessionLifetimeSec="sessionLifetimeSec"
    :maxBodySizeBytes="maxBodySize"
    :version="appVersion"
    @createNewUrl="newSessionHandler"
  ></main-header>
</template>

<script lang="ts">
import {defineComponent} from 'vue'
import MainHeader, {NewSessionSettings} from './components/main-header.vue'

export default defineComponent({
  components: {
    'main-header': MainHeader,
  },
  data(): {
    sessionUUID?: string
    sessionLifetimeSec: number
    maxBodySize: number
    appVersion: string
  } {
    return {
      sessionUUID: undefined,
      sessionLifetimeSec: Infinity,
      maxBodySize: 0, // in bytes
      appVersion: '0.0.0',
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
  },
  methods: {
    newSessionHandler(urlSettings: NewSessionSettings): void {
      console.log(urlSettings)
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
