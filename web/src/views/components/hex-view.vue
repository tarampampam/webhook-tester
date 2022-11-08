<template>
  <div class="container-fluid" v-if="content">
    <div
      v-if="content && content.length > 256"
      class="alert alert-secondary"
    >
      <p class="mb-0">
        HEX viewer may have performance problems with large request payloads. Please, make a
        <a href="https://github.com/tarampampam/webhook-tester/pulls" target="_blank">PR in the project
          repository</a> if you know how to solve this.
      </p>
    </div>

    <div class="row">
      <div class="col-xl-1 col-lg-2 d-none d-xl-block d-lg-block"></div>
      <div class="col-xl-5 col-lg-7 text-muted font-monospace text-nowrap">
        <span
          v-for="(colIdx, colNum) in bytesPerRow"
          v-bind:key="colIdx"
          :class="{ 'text-info': selectedCol === colNum }"
          class="me-2"
        >{{ numToHex(colNum, 2) }}</span>
      </div>
      <div class="col-xl-6 col-lg-3 d-none d-xl-block d-lg-block text-muted font-monospace">
        <span class="ps-5">ASCII</span>
      </div>
    </div>
    <div
      v-for="(bytesRow, rowNum) in lines()"
      v-bind:key="rowNum"
      class="row"
    >
      <div class="col-xl-1 col-lg-2 d-none d-xl-block d-lg-block font-monospace text-muted">
        <span :class="{ 'text-info': selectedRow === rowNum }">{{ numToHex(rowNum, 7) }}0</span>
      </div>
      <div class="col-xl-5 col-lg-7 font-monospace">
        <span
          v-for="(byte, byteNum) in bytesRow"
          v-bind:key="byteNum"
          class="me-2"
        >
        <span
          :class="{ 'text-warning': selectedRow === rowNum && selectedCol === byteNum }"
          @mouseover="handleChangeSelected(rowNum, byteNum)"
        >{{ numToHex(byte, 2) }}</span>
        </span>
      </div>
      <div class="col-xl-6 col-lg-3 d-none d-xl-block d-lg-block font-monospace">
        <span
          v-for="(byte, byteNum) in bytesRow"
          v-bind:key="byteNum"
          :class="{ 'text-warning': selectedRow === rowNum && selectedCol === byteNum }"
          @mouseover="handleChangeSelected(rowNum, byteNum)"
        >{{ byteToASCII(byte) }}</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent} from 'vue'

export default defineComponent({
  props: {
    content: { // note: do not forget to change watcher name!
      type: Uint8Array,
      default: undefined,
    },
  },

  data(): {
    bytesPerRow: number
    selectedRow: number
    selectedCol: number
  } {
    return {
      bytesPerRow: 8 * 2,
      selectedRow: -1,
      selectedCol: -1,
    }
  },

  watch: {
    content(): void {
      // reset own state
      this.selectedRow = -1
      this.selectedCol = -1
    },
  },

  methods: {
    lines(): Uint8Array[] {
      if (this.content) {
        const result: Uint8Array[] = []

        for (let i = 0; i < this.content.length; i += this.bytesPerRow) {
          result.push(this.content.slice(i, i + this.bytesPerRow))
        }

        return result
      }

      return []
    },

    numToHex(n: number, zerosCount: number): string {
      return String('0').repeat(zerosCount).concat(
        n.toString(16).toUpperCase()
      ).slice(-zerosCount)
    },

    byteToASCII(n: number): string {
      if (n >= 20 && n <= 126) { // is printable char?
        return String.fromCharCode(n)
      }

      return '·'
    },

    handleChangeSelected(row: number, col: number) {
      this.selectedRow = row
      this.selectedCol = col
    },
  }
})
</script>

<style lang="scss" scoped>
</style>
