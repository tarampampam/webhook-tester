<template>
    <div class="row request-details">
        <div class="col-md-12 col-lg-5 col-xl-4">
            <div class="row">
                <div class="col-7">
                    <h4>Request Details</h4>
                </div>
                <div class="col-5 text-right">
                    <button class="btn btn-primary btn-sm"
                            v-bind:data-clipboard-text="permalink"
                            target="_blank"
                            type="button"
                    >Copy permalink
                    </button>
                </div>
            </div>
            <div class="row mx-0">
                <table class="table table-hover table-sm table-borderless">
                    <tbody>
                    <tr>
                        <td>URL</td>
                        <td><code><a :href="getRequestURI">{{ getRequestURI }}</a></code>
                        </td>
                    </tr>
                    <tr>
                        <td>Method</td>
                        <td><code>{{ request.method.toUpperCase() }}</code></td>
                    </tr>
                    <tr>
                        <td>Client address</td>
                        <td><code>{{ request.client_address }}</code></td>
                    </tr>
                    <tr>
                        <td>Date</td>
                        <td><code>{{ formattedWhen }}</code></td>
                    </tr>
                    <tr>
                        <td>ID</td>
                        <td><code>{{ uuid }}</code></td>
                    </tr>
                    </tbody>
                </table>
            </div>
        </div>
        <div class="col-md-12 col-lg-7 col-xl-8" v-if="this.request.headers">
            <h4>Headers</h4>
            <div class="row mx-0">
                <table class="table table-hover table-sm table-borderless">
                    <tbody>
                    <tr v-for="(value, name) in this.request.headers">
                        <td>{{ name }}</td>
                        <td class="text-break"><code>{{ value }}</code></td>
                    </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</template>

<script>
    /* global module */

    'use strict';

    module.exports = {
        props: {
            request: {
                type: Object,
                default: null,
            },
            uuid: {
                type: String,
                default: null,
            },
        },

        data: function () {
            return {
                intervalId: null,
                formattedWhen: '',
                permalink: window.location.href,
                baseUrl: null,
            }
        },

        watch: {
            uuid: function () {
                this.updateFormattedWhen();
                this.permalink = window.location.href; // force update
            },
        },

        mounted: function () {
            this.baseURI = window.location.origin;
            this.updateFormattedWhen();
            this.intervalId = setInterval(() => this.updateFormattedWhen(), 1000);
        },

        computed: {
            /**
             * @returns {String}
             */
            getRequestURI: function () {
                let uri = typeof this.request.url === 'string'
                    ? this.request.url.replace(/^\/+/g, '')
                    : '...';

                return `${window.location.origin}/${uri}`;
            },
        },

        beforeDestroy: function () {
            clearInterval(this.intervalId);
        },

        methods: {
            updateFormattedWhen() {
                this.formattedWhen = this.request !== null && this.request.when != null
                    ? `${this.$moment(this.request.when).format('YYYY-MM-D h:mm:ss a')} (${this.$moment(this.request.when).fromNow()})`
                    : '';
            }
        }
    }
</script>

<style scoped>
    .request-details .text-break {
        word-break: break-all
    }
</style>
