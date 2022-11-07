import ClipboardJS from 'clipboard'

declare module '@vue/runtime-core' {
  // provide typings for `this.$clipboard` in Vue components
  interface ComponentCustomProperties {
    $clipboard: ClipboardJS // <https://clipboardjs.com/>
  }
}
