<template>
    <div>
        <main-header
            :current-web-hook-url="sessionRequestURI"
            :session-lifetime-sec="sessionLifetimeSec"
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
                            :uuid="r.uuid"
                            :client-address="r.client_address"
                            :method="r.method"
                            :when="r.when"
                            :key="r.uuid"
                            :class="{ active: requestUUID === r.uuid }"
                            @click.native="requestUUID = r.uuid"
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
                            <pre v-highlightjs="requestContent"><code class="javascript"></code></pre>
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
        },

        data: function () {
            return {
                /** @type {RecordedRequest[]} */
                requests: [],

                autoRequestNavigate: true,
                showRequestDetails: true,

                appVersion: null,
                sessionLifetimeSec: null,
                maxRequests: 50,
                maxBodySize: 0, // in bytes // TODO append this value into UI

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
                    this.maxRequests = settings.limits.max_requests;
                    this.sessionLifetimeSec = settings.limits.session_lifetime_sec;
                    this.maxBodySize = settings.limits.max_webhook_body_size;
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
             * @returns {String}
             */
            requestContent: function () {
                const request = this.getRequestByUUID(this.requestUUID);

                if (typeof request === 'object' && Object.prototype.hasOwnProperty.call(request, "content") && request.content !== '') {
                    return request.content;
                }

                return '// empty request body';
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
             * @returns {RecordedRequest|undefined}
             */
            getRequestByUUID(uuid) {
                if (typeof uuid === 'string' && this.requests.length > 0) {
                    for (let i = 0; i < this.requests.length; i++) {
                        if (this.requests[i].uuid === uuid) {
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
                        if (this.requests[i].uuid === uuid) {
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
                        if (this.requests[i].uuid === this.requestUUID) {
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
             * @param {APIRecordedRequest} rawRequest
             *
             * @returns {RecordedRequest}
             */
            formatRequestObject(rawRequest) {
                return {
                    uuid: rawRequest.uuid,
                    client_address: rawRequest.client_address,
                    method: rawRequest.method.toLowerCase(),
                    when: new Date(rawRequest.created_at_unix * 1000),
                    content: rawRequest.content,
                    headers: rawRequest.headers,
                    url: rawRequest.url,
                }
            },

            /**
             * @returns {Promise<undefined>}
             */
            reloadRequests() {
                return new Promise((resolve, reject) => {
                    this.$api.getAllSessionRequests(this.sessionUUID)
                        .then((requests) => {
                            this.requests.splice(0, this.requests.length); // make clear
                            requests.forEach((request) => this.requests.push(this.formatRequestObject(request)));
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
                            this.sessionUUID = newSessionData.uuid;
                            this.$session.setLocalSessionUUID(newSessionData.uuid);

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

                if (first !== undefined && first.uuid !== this.requestUUID) {
                    this.requestUUID = first.uuid;
                }
            },
            navigatePreviousRequest() {
                const current = this.getCurrentRequestIndex(), prev = this.requests[current - 1];

                if (prev !== undefined && prev.uuid !== this.requestUUID) {
                    this.requestUUID = prev.uuid;
                }
            },
            navigateNextRequest() {
                const current = this.getCurrentRequestIndex(), next = this.requests[current + 1];

                if (next !== undefined && next.uuid !== this.requestUUID) {
                    this.requestUUID = next.uuid;
                }
            },
            navigateLastRequest() {
                const last = this.requests[this.requests.length - 1];

                if (last !== undefined && last.uuid !== this.requestUUID) {
                    this.requestUUID = last.uuid;
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
             * @param {NewSessionData} urlSettings
             */
            newSessionHandler(urlSettings) {
                this.$api.startNewSession({
                    content_type: urlSettings.contentType,
                    status_code: urlSettings.statusCode,
                    response_delay: urlSettings.responseDelay,
                    response_body: urlSettings.responseBody,
                })
                    .then((newSessionData) => {
                        if (urlSettings.destroyCurrentSession === true) {
                            this.$api.deleteSession(this.sessionUUID)
                                .catch((err) => this.$izitoast.error({title: `Cannot destroy current session: ${err.message}`}))
                        }

                        this.sessionUUID = newSessionData.uuid;
                        this.$session.setLocalSessionUUID(newSessionData.uuid);

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
                        this.requests.unshift(this.formatRequestObject(request))

                        if (this.requestUUID === null || this.autoRequestNavigate === true) {
                            this.navigateFirstRequest();
                        }
                    })
                    .catch((err) => this.$izitoast.error({title: `Cannot load request with UUID ${requestUUID}: ${err.message}`}))
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
