<template>
  <div
    class="container-fluid"
    v-if="content"
  >
    <div class="row">
      <div class="d-none d-xl-block d-lg-block row-number" />
      <div class="text-muted font-monospace text-nowrap row-bytes">
        <span
          v-for="(colIdx, colNum) in bytesPerRow"
          :key="colIdx"
          :class="{ 'text-info': selectedCol === colNum, 'pe-2': colNum % 2 === 1 }"
          class="me-2"
        >{{ numToHex(colNum, 2) }}</span>
      </div>
      <div class="d-none d-xl-block d-lg-block text-muted text-center font-monospace row-ascii">
        ASCII
      </div>
    </div>
    <div
      v-for="(bytesRow, rowNum) in lines"
      :key="rowNum"
      class="row"
    >
      <div class="d-none d-xl-block d-lg-block font-monospace text-muted row-number">
        <span :class="{ 'text-info': selectedRow === rowNum }">{{ numToHex(rowNum, 7) }}0:</span>
      </div>
      <div class="font-monospace row-bytes">
        <span
          v-for="(byte, byteNum) in bytesRow"
          :key="byteNum"
          :class="{ 'pe-2': byteNum % 2 === 1 }"
          class="me-2"
        >
          <span
            :class="{ 'text-warning': selectedRow === rowNum && selectedCol === byteNum }"
            @mouseover="handleChangeSelected(rowNum, byteNum)"
          >{{ numToHex(byte, 2) }}</span>
        </span>
      </div>
      <div class="d-none d-xl-block d-lg-block font-monospace row-ascii">
        <span
          v-for="(byte, byteNum) in bytesRow"
          :key="byteNum"
          :class="{ 'text-warning': selectedRow === rowNum && selectedCol === byteNum }"
          @mouseover="handleChangeSelected(rowNum, byteNum)"
        >{{ byteToASCII(byte) }}</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent} from 'vue'

const extendedAsciiCodes = [
  128, // €
  130, // ‚
  131, // ƒ
  132, // „
  133, // …
  134, // †
  135, // ‡
  136, // ˆ
  137, // ‰
  139, // ‹
  145, // ‘
  146, // ’
  147, // “
  148, // ”
  149, // •
  150, // –
  151, // —
  152, // ˜
  153, // ™
  155, // ›
  156, // œ
  160, //
  162, // ¢
  163, // £
  165, // ¥
  167, // §
  169, // ©
  171, // «
  174, // ®
  187, // »
]

export default defineComponent({
  props: {
    content: { // note: do not forget to change watcher name!
      type: Uint8Array,
      default: undefined,
    },
  },

  data(): {
    lines: Uint8Array[]

    bytesPerRow: number
    selectedRow: number
    selectedCol: number
  } {
    return {
      lines: [],

      bytesPerRow: 8 * 2,
      selectedRow: -1,
      selectedCol: -1,
    }
  },

  mounted() {
    this.lines = this.content
      ? this.splitToLines(this.content)
      : []
  },

  watch: {
    content(): void{
      // reset own state
      this.selectedRow = -1
      this.selectedCol = -1

      this.lines = this.content
        ? this.splitToLines(this.content)
        : []
    },
  },

  methods: {
    splitToLines(src: Uint8Array): Uint8Array[] {
      const result: Uint8Array[] = []

      for (let i = 0; i < src.length; i += this.bytesPerRow) {
        result.push(src.slice(i, i + this.bytesPerRow))
      }

      return result
    },

    numToHex(n: number, zerosCount: number): string {
      return String('0').repeat(zerosCount).concat(
        n.toString(16).toUpperCase()
      ).slice(-zerosCount)
    },

    byteToASCII(n: number): string {
      if ((n >= 32 && n <= 126) || extendedAsciiCodes.includes(n)) { // is printable char?
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
.row-number {
  width: 110px;
}

.row-bytes {
  width: 515px;
}

.row-ascii {
  width: 190px;
}

.row-bytes, .row-ascii {
  cursor: default;
}
</style>
