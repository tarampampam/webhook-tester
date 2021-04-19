<template>
    <div class="row request-details">
        <div class="col-md-12 col-lg-5 col-xl-4">
            <div class="row">
                <div class="col-7">
                    <h4>Request details</h4>
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
                    <span class="badge text-uppercase"
                          :class="methodClass"
                    >{{ request.method.toUpperCase() }}</span>
                </div>
            </div>

            <div class="row pb-1">
                <div class="col-lg-3 text-lg-right">From</div>
                <div class="col-lg-9">
                    <a :href="'https://who.is/whois-ip/ip-address/' + request.clientAddress"
                       target="_blank"
                       rel="noreferrer"
                       title="WhoIs?"
                    >
                        <strong>{{ request.clientAddress }}</strong>
                    </a>
                </div>
            </div>

            <div class="row pb-1">
                <div class="col-lg-3 text-lg-right">When</div>
                <div class="col-lg-9">
                    <span>{{ formattedWhen }}</span>
                </div>
            </div>

            <div class="row pb-1">
                <div class="col-lg-3 text-lg-right">Size</div>
                <div class="col-lg-9">
                    <span v-if="contentLength">{{ contentLength }} bytes</span>
                    <span v-else class="text-muted">&mdash;</span>
                </div>
            </div>

            <div class="row pb-1">
                <div class="col-lg-3 text-lg-right">ID</div>
                <div class="col-lg-9 text-break">
                    <code>{{ uuid }}</code>
                </div>
            </div>
        </div>

        <div class="col-md-12 col-lg-7 col-xl-8 mt-3 mt-md-3 mt-lg-0" v-if="this.request.headers">
            <h4>Headers</h4>
            <div v-for="(header) in this.request.headers"
                 :key="header.name"
                 class="row pb-1">
                <div class="col-lg-4 col-xl-3 text-lg-right">
                    {{ header.name }}
                </div>
                <div class="col-lg-8 col-xl-9 text-break">
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
                let uri = (typeof this.request === 'object' && this.request !== null && typeof this.request.url === 'string')
                    ? this.request.url.replace(/^\/+/g, '')
                    : '...';

                return `${window.location.origin}/${uri}`;
            },

            methodClass: function () {
                if (typeof this.request === 'object' && this.request !== null && typeof this.request.method === 'string') {
                    switch (this.request.method.toLowerCase()) {
                        case 'get':
                            return 'badge-success';
                        case 'post':
                        case 'put':
                            return 'badge-info';
                        case 'delete':
                            return 'badge-danger';
                    }
                }

                return 'badge-light';
            },

            /**
             * @returns {Number}
             */
            contentLength() {
                if (typeof this.request === 'object' && this.request !== null) {
                    return this.request.content.length;
                }

                return 0;
            },
        },

        beforeDestroy: function () {
            clearInterval(this.intervalId);
        },

        methods: {
            updateFormattedWhen() {
                this.formattedWhen = this.request !== null && this.request.createdAt != null
                    ? `${this.$moment(this.request.createdAt).format('YYYY-MM-D h:mm:ss a')} (${this.$moment(this.request.createdAt).fromNow()})`
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
