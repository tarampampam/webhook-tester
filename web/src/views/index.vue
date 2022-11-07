<template>
  <main-header
    :current-web-hook-url="sessionRequestURI"
    :session-lifetime-sec="sessionLifetimeSec"
    :max-body-size-bytes="maxBodySize"
    :version="appVersion"
    @on-new-url="newSessionHandler"
  ></main-header>
</template>

<script lang="ts">
import {defineComponent} from 'vue'
import MainHeader from './components/main-header.vue'

export default defineComponent({
  components: {
    'main-header': MainHeader,
  },
  data() {
    return {}
  },
  methods: {},
  created(): void {
    //
  },
  mounted() {
    // hide main loading spinner
    document.getElementById('main-loader')?.remove()
  },
})
</script>

<style lang="scss">
@import "~bootswatch/dist/darkly/variables";
@import "~bootstrap/scss/bootstrap";
@import "~bootswatch/dist/darkly/bootswatch";
@import "~izitoast/dist/css/iziToast";
</style>
