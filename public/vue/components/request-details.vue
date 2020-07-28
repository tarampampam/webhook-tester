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
                            type="button"
                    >Copy permalink
                    </button>
                </div>
            </div>

            <div class="row pb-1">
                <div class="col-lg-3 text-lg-right">URL</div>
                <div class="col-lg-9 text-break">
                    <code><a :href="getRequestURI">{{ getRequestURI }}</a></code>
                </div>
            </div>

            <div class="row pb-1">
                <div class="col-lg-3 text-lg-right">Method</div>
                <div class="col-lg-9">
                    <code>{{ request.method.toUpperCase() }}</code>
                </div>
            </div>

            <div class="row pb-1">
                <div class="col-lg-3 text-lg-right">From</div>
                <div class="col-lg-9">
                    <code>{{ request.client_address }}</code>
                </div>
            </div>

            <div class="row pb-1">
                <div class="col-lg-3 text-lg-right">When</div>
                <div class="col-lg-9">
                    <code>{{ formattedWhen }}</code>
                </div>
            </div>

            <div class="row pb-1">
                <div class="col-lg-3 text-lg-right">ID</div>
                <div class="col-lg-9">
                    <code>{{ uuid }}</code>
                </div>
            </div>
        </div>

        <div class="col-md-12 col-lg-7 col-xl-8 mt-3 mt-md-3 mt-lg-0" v-if="this.request.headers">
            <h4>Headers</h4>
            <div v-for="(header) in this.request.headers"
                 class="row pb-1">
                <div class="col-lg-4 col-xl-2 text-lg-right">
                    {{ header.name }}
                </div>
                <div class="col-lg-8 col-xl-10 text-break">
                    <code>{{ header.value }}</code>
                </div>
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
            }
        },

        watch: {
            uuid: function () {
                this.updateFormattedWhen();
                this.permalink = window.location.href; // force update
            },
        },

        mounted: function () {
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
        word-break: break-all;
    }
</style>
