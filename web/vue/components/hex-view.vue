<template functional>
    <div class="container-fluid" v-if="content">
        <div
            v-if="content !== undefined && content.length > 256"
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
            <div class="col-xl-5 col-lg-7 text-muted text-monospace text-nowrap">
                <span
                    v-for="(colIdx, colNum) in bytesPerRow"
                    v-bind:key="colIdx"
                    :class="{ 'text-info': selectedCol === colNum }"
                    class="mr-2"
                >{{ numToHex(colNum) | formatHex(2) }}</span>
            </div>
            <div class="col-xl-6 col-lg-3 d-none d-xl-block d-lg-block text-muted text-monospace">
                <span class="pl-5">ASCII</span>
            </div>
        </div>
        <div
            v-for="(bytesRow, rowNum) in lines(content)"
            v-bind:key="rowNum"
            class="row"
        >
            <div class="col-xl-1 col-lg-2 d-none d-xl-block d-lg-block text-monospace text-muted">
                <span :class="{ 'text-info': selectedRow === rowNum }">{{ numToHex(rowNum) | formatHex(7) }}0</span>
            </div>
            <div class="col-xl-5 col-lg-7 text-monospace">
                <span
                    v-for="(byte, byteNum) in bytesRow"
                    v-bind:key="byteNum"
                    class="mr-2"
                >
                    <span
                        :class="{ 'text-warning': selectedRow === rowNum && selectedCol === byteNum }"
                        @mouseover="handleChangeSelected(rowNum, byteNum)"
                    >{{ numToHex(byte) | formatHex(2) }}</span>
                </span>
            </div>
            <div class="col-xl-6 col-lg-3 d-none d-xl-block d-lg-block text-monospace">
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

<script>
    /* global module */

    'use strict';

    module.exports = {
        props: {
            content: { // note: do not forget to change watcher name!
                type: Uint8Array,
                default: undefined,
            },
        },

        data: function () {
            return {
                bytesPerRow: 8 * 2,
                selectedRow: -1,
                selectedCol: -1,
            }
        },

        watch: {
            content() {
                // reset own state
                this.selectedRow = -1;
                this.selectedCol = -1;
            },
        },

        filters: {
            /**
             * @param  {String} s
             * @param  {Number} zerosCount
             * @return {String}
             */
            formatHex(s, zerosCount) {
                return String('0').repeat(zerosCount).concat(s).slice(-zerosCount);
            },
        },

        methods: {
            /**
             * @param {ArrayLike} arr
             * @return {ArrayLike}
             */
            lines(arr) {
                return this.content.reduce((rows, key, index) => (
                        index % this.bytesPerRow === 0
                            ? rows.push([key])
                            : rows[rows.length-1].push(key)
                    ) && rows, []
                );
            },

            /**
             * @param {Number} n
             * @return {string}
             */
            numToHex(n) {
                return n.toString(16).toUpperCase();
            },

            /**
             * @param {Number} n
             * @return {string}
             */
            byteToASCII(n) {
                if (n >= 20 && n <= 126) { // is printable char?
                    return String.fromCharCode(n);
                }

                return '·';
            },

            /**
             * @param {Number} row
             * @param {Number} col
             */
            handleChangeSelected(row, col) {
                this.selectedRow = row;
                this.selectedCol = col;
            },
        }
    }
</script>

<style scoped>
</style>
