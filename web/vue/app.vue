<template>
    <div>
        <main-header
            :current-web-hook-url="sessionRequestURI"
            :session-lifetime-sec="sessionLifetimeSec"
            :max-body-size-bytes="maxBodySize"
            :version="appVersion"
            @on-new-url="newSessionHandler"
        ></main-header>

        <div class="container-fluid mb-2">
            <div class="row flex-xl-nowrap">
                <div class="sidebar col-sm-5 col-md-4 col-lg-3 col-xl-2 px-2 py-0">
                    <div class="pl-3 pt-4 pr-3 pb-3">
                        <div class="d-flex w-100 justify-content-between">
                            <h5 class="text-uppercase mb-0">Requests <span
                                class="badge badge-primary badge-pill total-requests-count"
                            >{{ requests.length }}</span></h5>
                            <button type="button"
                                    class="btn btn-outline-danger btn-sm position-relative button-delete-all"
                                    v-if="requests.length > 0"
                                    @click="deleteAllRequestsHandler">Delete all
                            </button>
                        </div>
                    </div>

                    <div class="list-group" v-if="requests.length > 0">
                        <request-plate
                            v-for="r in this.requests"
                            class="request-plate"
                            :uuid="r.UUID"
                            :client-address="r.clientAddress"
                            :method="r.method"
                            :when="r.createdAt"
                            :key="r.UUID"
                            :class="{ active: requestUUID === r.UUID }"
                            @click.native="requestUUID = r.UUID"
                            @on-delete="deleteRequestHandler"
                        ></request-plate>
                    </div>
                    <div v-else class="text-muted text-center mt-3">
                        <span class="spinner-border spinner-border-sm mr-1"></span> Waiting for first request
                    </div>
                </div>

                <div class="col-sm-7 col-md-8 col-lg-9 col-xl-10 py-3 pl-md-4" role="main">
                    <div v-if="requests.length > 0 && this.requestUUID !== null">
                        <div class="row pt-2">
                            <div class="col-6">
                                <div class="btn-group pb-1" role="group">
                                    <button type="button" class="btn btn-secondary btn-sm"
                                            @click.prevent="navigateFirstRequest"
                                            :class="{disabled: requests.length <= 1}"
                                    >
                                        First request
                                    </button>
                                    <button type="button" class="btn btn-secondary btn-sm"
                                            @click="navigatePreviousRequest"
                                            :class="{disabled: requests.length <= 1 || this.requestUUID === null}"
                                    >
                                        <i class="fas fa-arrow-left pr-1"></i>Previous
                                    </button>
                                </div>
                                <div class="btn-group pb-1" role="group">
                                    <button type="button" class="btn btn-secondary btn-sm"
                                            @click="navigateNextRequest"
                                            :class="{disabled: requests.length <= 1 || this.requestUUID === null}"
                                    >
                                        Next<i class="fas fa-arrow-right pl-1"></i>
                                    </button>
                                    <button type="button" class="btn btn-secondary btn-sm"
                                            @click="navigateLastRequest"
                                            :class="{disabled: requests.length <= 1}"
                                    >
                                        Last request
                                    </button>
                                </div>
                            </div>
                            <div class="col-6 pb-1 text-right">
                                <div class="custom-control custom-checkbox d-inline-block">
                                    <input type="checkbox"
                                           class="custom-control-input"
                                           id="show-details"
                                           v-model="showRequestDetails">
                                    <label class="custom-control-label" for="show-details">Show details</label>
                                </div>
                                <div class="custom-control custom-checkbox d-inline-block ml-3"
                                     title="Automatically select and go to the latest incoming webhook request">
                                    <input type="checkbox"
                                           class="custom-control-input"
                                           id="auto-navigate"
                                           v-model="autoRequestNavigate">
                                    <label class="custom-control-label" for="auto-navigate">Auto navigate</label>
                                </div>
                            </div>
                        </div>

                        <request-details
                            v-if="showRequestDetails"
                            class="pt-3"
                            :request="getRequestByUUID(this.requestUUID)"
                            :uuid="this.requestUUID"
                        ></request-details>

                        <div class="pt-3">
                            <h4>Request body</h4>

                            <div v-if="requestContentExists">
                                <ul class="nav nav-pills">
                                    <li class="nav-item">
                                        <span
                                            class="btn nav-link pl-4 pr-4 pt-1 pb-1"
                                            :class="{ 'active': requestContentViewMode === 'text' }"
                                            @click="requestContentViewMode='text'"
                                        >
                                            <i class="fas fa-font"></i> Text view
                                        </span>
                                    </li>
                                    <li class="nav-item">
                                        <span
                                            class="btn nav-link pl-4 pr-4 pt-1 pb-1"
                                            :class="{ 'active': requestContentViewMode === 'binary' }"
                                            @click="requestContentViewMode='binary'"
                                        >
                                            <i class="fas fa-atom"></i> Binary view
                                        </span>
                                    </li>
                                    <li
                                        class="nav-item"
                                        v-if="getRequestByUUID(this.requestUUID) !== undefined"
                                    >
                                        <span
                                            class="btn nav-link pl-4 pr-4 pt-1 pb-1"
                                            @click="handleDownloadRequestContent"
                                        >
                                            <i class="fas fa-download"></i> Download
                                        </span>
                                    </li>
                                </ul>
                                <div class="tab-content pt-2 pb-2">
                                    <div
                                        class="tab-pane active"
                                        v-if="requestContentViewMode === 'text'"
                                    >
                                        <pre v-highlightjs="requestContent"><code></code></pre>
                                    </div>
                                    <div
                                        class="tab-pane active pt-2"
                                        v-if="requestContentViewMode === 'binary'"
                                    >
                                        <hex-view
                                            :content="requestBinaryContent"
                                        ></hex-view>
                                    </div>
                                </div>
                            </div>
                            <div v-else class="pt-1 pb-1">
                                <p class="text-muted small text-monospace">// empty request body</p>
                            </div>
                        </div>
                    </div>
                    <index-empty
                        v-else
                        :current-web-hook-url="sessionRequestURI"
                    ></index-empty>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
    /* global module */

    'use strict';

    const textDecoder = new TextDecoder("utf-8");

    module.exports = {
        /**
         * Force the Vue instance to re-render. Note it does not affect all child components, only the instance
         * itself and child components with inserted slot content.
         *
         * @method
         * @name $forceUpdate
         */

        components: {
            'main-header': 'url:/vue/components/main-header.vue',
            'request-plate': 'url:/vue/components/request-plate.vue',
            'request-details': 'url:/vue/components/request-details.vue',
            'index-empty': 'url:/vue/components/index-empty.vue',
            'hex-view': 'url:/vue/components/hex-view.vue',
        },

        data: function () {
            return {
                /** @type {APIRecordedRequest[]} */
                requests: [],

                autoRequestNavigate: true,
                showRequestDetails: true,

                requestContentViewMode: 'text', // or 'binary'

                appVersion: null,
                sessionLifetimeSec: null,
                maxRequests: 50,
                maxBodySize: 0, // in bytes

                sessionUUID: null,
                requestUUID: null,

                ws: null,
            }
        },

        created() {
            this.$api.getAppVersion()
                .then((ver) => this.appVersion = ver.version);

            this.$api.getAppSettings()
                .then((settings) => {
                    this.maxRequests = settings.limits.maxRequests;
                    this.sessionLifetimeSec = settings.limits.sessionLifetimeSec;
                    this.maxBodySize = settings.limits.maxWebhookBodySize;
                });

            this.wsRefreshConnection();

            this.initSession();
            this.initRequest();
        },

        mounted() {
            // hide main loading spinner
            window.setTimeout(() => document.getElementById('main-loader').remove(), 150);
        },

        computed: {
            /**
             * @returns {String}
             */
            sessionRequestURI: function () {
                let uuid = this.sessionUUID === null
                    ? '________-____-____-____-____________'
                    : this.sessionUUID;

                return `${window.location.origin}/${uuid}`;
            },

            /**
             * @returns {Boolean}
             */
            requestContentExists: function () {
                const request = this.getRequestByUUID(this.requestUUID);

                return request !== undefined && request.content.length > 0;
            },

            /**
             * @returns {String}
             */
            requestContent: function () {
                const request = this.getRequestByUUID(this.requestUUID);

                if (request !== undefined && request.content.length > 0) {
                    return textDecoder.decode(request.content);
                }

                return '';
            },

            /**
             * @returns {Uint8Array}
             */
            requestBinaryContent: function () {
                const request = this.getRequestByUUID(this.requestUUID);

                if (request !== undefined && request.content.length > 0) {
                    return request.content;
                }

                return new Uint8Array(0);
            },
        },

        watch: {
            sessionUUID: function () {
                if (this.$route.params.sessionUUID !== this.sessionUUID) {
                    this.$router.push({
                        name: 'request', params: {
                            sessionUUID: this.sessionUUID,
                        }
                    });
                }

                this.wsRefreshConnection();
            },
            requestUUID: function () {
                if (this.$route.params.requestUUID !== this.requestUUID) {
                    this.$router.push({
                        name: 'request', params: {
                            sessionUUID: this.sessionUUID,
                            requestUUID: this.requestUUID,
                        }
                    }).catch(() => {
                        // do nothing
                    });
                }
            },
            requests: function () {
                // limit maximal requests length
                if (this.requests.length > this.maxRequests) {
                    this.requests.splice(this.maxRequests, this.requests.length);

                    if (this.getRequestByUUID(this.requestUUID) === undefined) {
                        this.requestUUID = null;
                    }
                }
            },
        },

        methods: {
            wsRefreshConnection() {
                const requestRegistered = 'request-registered',
                    requestDeleted = 'request-deleted',
                    requestsDeleted = 'requests-deleted';

                // unsubscribe first
                if (this.ws !== null) {
                    this.ws.close(); // docs: <https://github.com/joewalnes/reconnecting-websocket#wsclosecode-reason>
                    this.ws = null;
                }

                if (this.sessionUUID !== null) {
                    this.ws = this.$ws.newRenewableSessionConnection(this.sessionUUID, (name, data) => {
                        // route incoming events
                        switch (name) {
                            case requestRegistered:
                                this.wsRegisteredRequestHandler(data);
                                break;

                            case requestDeleted:
                                this.deleteRequest(data);
                                break;

                            case requestsDeleted:
                                this.clearRequests();
                                break;
                        }
                    });
                }
            },

            /**
             * @param {String} uuid
             * @returns {APIRecordedRequest|undefined}
             */
            getRequestByUUID(uuid) {
                if (typeof uuid === 'string' && this.requests.length > 0) {
                    for (let i = 0; i < this.requests.length; i++) {
                        if (this.requests[i].UUID === uuid) {
                            return this.requests[i];
                        }
                    }
                }

                return undefined;
            },

            /**
             * @param {String} uuid
             * @returns {Number|undefined}
             */
            getRequestIndexByUUID(uuid) {
                if (typeof uuid === 'string' && this.requests.length > 0) {
                    for (let i = 0; i < this.requests.length; i++) {
                        if (this.requests[i].UUID === uuid) {
                            return i;
                        }
                    }
                }

                return undefined;
            },

            /**
             * @returns {Number|undefined}
             */
            getCurrentRequestIndex() {
                if (this.requests.length > 0) {
                    for (let i = 0; i < this.requests.length; i++) {
                        if (this.requests[i].UUID === this.requestUUID) {
                            return i;
                        }
                    }
                }

                return undefined;
            },

            /**
             * @param {String} uuid
             * @returns {Boolean}
             */
            isValidUUID(uuid) {
                return typeof uuid === 'string' && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(uuid);
            },

            /**
             * @returns {Promise<undefined>}
             */
            reloadRequests() {
                return new Promise((resolve, reject) => {
                    this.$api.getAllSessionRequests(this.sessionUUID)
                        .then((requests) => {
                            this.requests.splice(0, this.requests.length); // make clear
                            requests.forEach((request) => this.requests.push(request));
                            resolve();
                        })
                        .catch((err) => reject(err))
                });
            },

            initSession() {
                const localSessionUUID = this.$session.getLocalSessionUUID();
                const routeSessionUUID = this.$route.params.sessionUUID;
                const sessionUUID = this.isValidUUID(routeSessionUUID)
                    ? routeSessionUUID
                    : (this.isValidUUID(localSessionUUID) ? localSessionUUID : null);

                const startNewSession = () => {
                    this.$api.startNewSession({})
                        .then((newSessionData) => {
                            this.sessionUUID = newSessionData.UUID;
                            this.$session.setLocalSessionUUID(newSessionData.UUID);

                            this.reloadRequests()
                                .catch((err) => this.$izitoast.error({title: `Cannot retrieve requests: ${err.message}`}))
                        })
                        .catch((err) => this.$izitoast.error({title: `Cannot create new session: ${err.message}`}))
                };

                if (sessionUUID !== null) {
                    this.sessionUUID = sessionUUID;

                    this.reloadRequests()
                        .then(() => {
                            if (this.requestUUID === null || this.getRequestIndexByUUID(this.requestUUID) === undefined) {
                                this.navigateFirstRequest();
                            }
                        })
                        .catch(() => startNewSession())
                } else {
                    startNewSession()
                }
            },

            initRequest() {
                const routeRequestUUID = this.$route.params.requestUUID;

                if (this.isValidUUID(routeRequestUUID)) {
                    this.requestUUID = routeRequestUUID;
                }
            },

            navigateFirstRequest() {
                const first = this.requests[0];

                if (first !== undefined && first.UUID !== this.requestUUID) {
                    this.requestUUID = first.UUID;
                }
            },

            navigatePreviousRequest() {
                const current = this.getCurrentRequestIndex(), prev = this.requests[current - 1];

                if (prev !== undefined && prev.UUID !== this.requestUUID) {
                    this.requestUUID = prev.UUID;
                }
            },

            navigateNextRequest() {
                const current = this.getCurrentRequestIndex(), next = this.requests[current + 1];

                if (next !== undefined && next.UUID !== this.requestUUID) {
                    this.requestUUID = next.UUID;
                }
            },

            navigateLastRequest() {
                const last = this.requests[this.requests.length - 1];

                if (last !== undefined && last.UUID !== this.requestUUID) {
                    this.requestUUID = last.UUID;
                }
            },

            /**
             * @param {String} uuid
             */
            deleteRequest(uuid) {
                const currentIndex = this.getRequestIndexByUUID(uuid);

                if (currentIndex !== undefined) {
                    if (uuid !== this.requestUUID) {
                        // do nothing
                    } else if (this.requests[currentIndex + 1] !== undefined) {
                        this.navigateNextRequest();
                    } else if (this.requests[currentIndex - 1] !== undefined) {
                        this.navigatePreviousRequest();
                    }

                    this.requests.splice(currentIndex, 1); // remove request object from stack
                }
            },

            clearRequests() {
                this.requests.splice(0, this.requests.length);
                this.requestUUID = null;
            },

            deleteAllRequestsHandler() {
                this.$api.deleteAllSessionRequests(this.sessionUUID)
                    .then((status) => {
                        if (status.success === true) {
                            this.$izitoast.success({title: 'All requests successfully removed!'});
                        } else {
                            throw new Error(`I've got unsuccessful status`);
                        }
                    })
                    .catch((err) => this.$izitoast.error({title: `Cannot remove all requests: ${err.message}`}))

                this.clearRequests();
            },

            /**
             * @param {String} uuid
             */
            deleteRequestHandler(uuid) {
                this.$api.deleteSessionRequest(this.sessionUUID, uuid)
                    .then((status) => {
                        if (status.success !== true) {
                            throw new Error(`Unsuccessful status returned`);
                        }
                    })
                    .catch((err) => this.$izitoast.error({title: `Cannot remove request: ${err.message}`}))

                this.deleteRequest(uuid);
            },

            /**
             * @typedef {Object} NewSessionSettings
             * @property {Number} [statusCode]
             * @property {String} [contentType]
             * @property {Number} [responseDelay]
             * @property {Uint8Array} [responseBody]
             * @property {Boolean} destroyCurrentSession
             */
            /**
             * @param {NewSessionSettings} urlSettings
             */
            newSessionHandler(urlSettings) {
                this.$api.startNewSession({
                    contentType: urlSettings.contentType,
                    statusCode: urlSettings.statusCode,
                    responseDelay: urlSettings.responseDelay,
                    responseContent: urlSettings.responseBody,
                })
                    .then((newSessionData) => {
                        if (urlSettings.destroyCurrentSession === true) {
                            this.$api.deleteSession(this.sessionUUID)
                                .catch((err) => this.$izitoast.error({title: `Cannot destroy current session: ${err.message}`}))
                        }

                        this.sessionUUID = newSessionData.UUID;
                        this.$session.setLocalSessionUUID(newSessionData.UUID);

                        this.clearRequests();
                        this.$izitoast.success({title: 'New session started!'});
                    })
                    .catch((err) => this.$izitoast.error({title: `Cannot create new session: ${err.message}`}))
            },

            /**
             * @param {String} requestUUID
             */
            wsRegisteredRequestHandler(requestUUID) {
                this.$izitoast.info({
                    title: 'New request',
                    message: 'New incoming webhook request',
                    timeout: 2000,
                    closeOnClick: true
                });

                this.$api.getSessionRequest(this.sessionUUID, requestUUID)
                    .then((request) => {
                        // push at the first position
                        this.requests.unshift(request)

                        if (this.requestUUID === null || this.autoRequestNavigate === true) {
                            this.navigateFirstRequest();
                        }
                    })
                    .catch((err) => this.$izitoast.error({title: `Cannot load request with UUID ${requestUUID}: ${err.message}`}))
            },

            handleDownloadRequestContent() {
                const request = this.getRequestByUUID(this.requestUUID);

                if (request !== undefined && request.content.length > 0) {
                    const $body = document.body, $a = document.createElement('a');

                    $a.setAttribute('href', 'data:application/octet-stream;charset=utf-8,' + encodeURIComponent(textDecoder.decode(request.content)));
                    $a.setAttribute('download', this.requestUUID + '.bin');
                    $a.style.display = 'none';

                    $body.appendChild($a);
                    $a.click();
                    $body.removeChild($a);
                }
            },
        }
    }
</script>

<style scoped>
    .btn:focus,
    .btn:active {
        outline: none !important;
        box-shadow: none;
    }

    .total-requests-count {
        position: relative;
        top: -.15em;
    }

    .request-plate {
        cursor: pointer;
    }

    .button-delete-all {
        top: -2px;
    }

    .hljs {
        background-color: transparent;
    }
</style>
