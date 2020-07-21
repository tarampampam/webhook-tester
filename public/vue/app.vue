<template>
    <div>
        <main-header
            :current-web-hook-url="sessionRequestURI"
            :session-lifetime-sec="sessionLifetimeSec"
            :version="appVersion"
            @on-new-url="newUrlHandler"
        ></main-header>

        <div class="container-fluid">
            <div class="row flex-xl-nowrap">
                <div class="sidebar col-sm-5 col-md-4 col-lg-3 col-xl-2 px-2 py-0">
                    <div class="pl-3 pt-4 pr-3 pb-3">
                        <div class="d-flex w-100 justify-content-between">
                            <h5 class="text-uppercase mb-0">Requests <span
                                class="badge badge-primary badge-pill total-requests-count"
                            >{{ getRequestsCount() }}</span></h5>
                            <button type="button"
                                    class="btn btn-outline-danger btn-sm"
                                    v-if="getRequestsCount() > 0"
                                    @click="deleteAllRequests()">Delete all
                            </button>
                        </div>
                    </div>

                    <div class="list-group" v-if="getRequestsCount() > 0">
                        <request-plate
                            v-for="(request, uuid) in this.requests"
                            class="request-plate"
                            :uuid="uuid"
                            :ip="request.ip"
                            :method="request.method"
                            :when="request.when"
                            :key="uuid"
                            :class="{ active: requestUUID === uuid }"
                            @click.native="requestUUID = uuid"
                            @on-delete="deleteRequestHandler"
                        ></request-plate>
                    </div>
                    <div v-else class="text-muted text-center mt-3">
                        <span class="spinner-border spinner-border-sm mr-1"></span> Waiting for first request
                    </div>
                </div>

                <div class="col-sm-7 col-md-8 col-lg-9 col-xl-10 py-3 pl-md-4" role="main">
                    <div v-if="getRequestsCount() > 0">
                        <div class="row pt-2">
                            <div class="col-6">
                                <div class="btn-group pb-1" role="group">
                                    <button type="button" class="btn btn-secondary btn-sm"
                                            @click.prevent="navigateFirstRequest"
                                            :class="{disabled: getRequestsCount() <= 1}"
                                    >
                                        First request
                                    </button>
                                    <button type="button" class="btn btn-secondary btn-sm"
                                            @click="navigatePreviousRequest"
                                            :class="{disabled: getRequestsCount() <= 1}"
                                    >
                                        <i class="fas fa-arrow-left"></i> Previous
                                    </button>
                                </div>
                                <div class="btn-group pb-1" role="group">
                                    <button type="button" class="btn btn-secondary btn-sm"
                                            @click="navigateNextRequest"
                                            :class="{disabled: getRequestsCount() <= 1}"
                                    >
                                        Next <i class="fas fa-arrow-right"></i>
                                    </button>
                                    <button type="button" class="btn btn-secondary btn-sm"
                                            @click="navigateLastRequest"
                                            :class="{disabled: getRequestsCount() <= 1}"
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
                            :request="this.requests[this.requestUUID]"
                            :uuid="this.requestUUID"
                        ></request-details>

                        <div class="pt-3">
                            <h4>Body</h4>
                            <pre v-if="requestsContent">{{ requestsContent | prettyContent }}</pre>
                            <pre v-else class="text-muted">// empty request body</pre>
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
            let pastDate = new Date; // just for a test
            pastDate.setHours(pastDate.getHours() - 2);

            return {
                requests: {
                    // 'cd5e695f-1784-4dcf-9b3f-ef66c9a0aaaa': {
                    //     ip: '1.1.1.1',
                    //     method: 'post',
                    //     when: pastDate,
                    //     content: 'foo bar',
                    //     url: 'https://foo.example.com/aaaaaaaa-bbbb-cccc-dddd-000000000000/foobar',
                    //     hostname: 'some_host',
                    //     headers: {
                    //         "host": "foo.example.com",
                    //         "user-agent": "curl\/7.58.0",
                    //         "accept": "text\/html,application\/xhtml+xml",
                    //    },
                    // },
                },

                autoRequestNavigate: true,
                showRequestDetails: true,

                appVersion: null,
                sessionLifetimeSec: null,

                sessionUUID: null,
                requestUUID: null,
            }
        },

        created() {
            this.$api.getAppSettings()
                .then((settings) => {
                    if (settings.hasOwnProperty('version')) {
                        this.appVersion = settings.version;
                    }
                    this.sessionLifetimeSec = settings.limits.session_lifetime_sec;
                });

            this.initSession();
            this.initRequest();
        },

        mounted() {
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
             * @returns {String|null}
             */
            requestsContent: function () {
                const request = this.requests[this.requestUUID];

                if (typeof request === 'object' && request.hasOwnProperty('content') && request.content !== '') {
                    return request.content;
                }

                return null;
            },
        },

        filters: {
            prettyContent: function(value) {
                // is json?
                try {
                    return JSON.stringify(JSON.parse(value), null, 2);
                } catch (e) {
                    //
                }

                return value; // as-is
            }
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
            },
            requestUUID: function () {
                if (this.$route.params.requestUUID !== this.requestUUID) {
                    this.$router.push({
                        name: 'request', params: {
                            sessionUUID: this.sessionUUID,
                            requestUUID: this.requestUUID,
                        }
                    });
                }
            },
        },

        methods: {
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
                            for (let uuid in requests) {
                                if (requests.hasOwnProperty(uuid)) {
                                    let request = requests[uuid];

                                    this.requests[uuid] = {
                                        ip: request.ip,
                                        method: request.method.toLowerCase(),
                                        when: new Date(request.created_at_unix * 1000),
                                        content: request.content,
                                        url: request.url,
                                        hostname: request.hostname,
                                        headers: request.headers,
                                        __updated: true,
                                    };
                                }
                            }

                            // remove non-updated objects and (or) special `__updated` property
                            for (let uuid in this.requests) {
                                if (this.requests.hasOwnProperty(uuid)) {
                                    let request = this.requests[uuid];

                                    if (request.hasOwnProperty('__updated') && request.__updated === true) {
                                        delete request['__updated'];
                                    } else {
                                        delete this.requests[uuid];
                                    }
                                }
                            }

                            if (!this.requests.hasOwnProperty(this.requestUUID)) {
                                /** @var {String|undefined} uuidToSet */
                                const uuidToSet = Object.keys(this.requests)[0];

                                if (this.isValidUUID(uuidToSet)) {
                                    this.requestUUID = uuidToSet;
                                }
                            }

                            this.$forceUpdate();

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
                                .catch((err) => this.$izitoast.error({
                                    title: `Cannot retrieve requests: ${err.message}`,
                                    zindex: 10
                                }))
                        })
                        .catch((err) => this.$izitoast.error({
                            title: `Cannot create new session: ${err.message}`,
                            zindex: 10
                        }))
                };

                if (sessionUUID !== null) {
                    this.sessionUUID = sessionUUID;

                    this.reloadRequests()
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

            /**
             * @returns {Number}
             */
            getRequestsCount() {
                return Object.keys(this.requests).length;
            },

            navigateFirstRequest() {
                const firstUuid = Object.keys(this.requests)[0];
                console.log(111);
                if (firstUuid !== undefined && this.requestUUID !== firstUuid) {
                    this.requestUUID = firstUuid;
                }
            },
            navigatePreviousRequest() {
                const keys = Object.keys(this.requests), prev = keys[keys.indexOf(this.requestUUID) - 1];

                if (prev !== undefined && this.requestUUID !== prev) {
                    this.requestUUID = prev;
                }
            },
            navigateNextRequest() {
                const keys = Object.keys(this.requests), next = keys[keys.indexOf(this.requestUUID) + 1];

                if (next !== undefined && this.requestUUID !== next) {
                    this.requestUUID = next;
                }
            },
            navigateLastRequest() {
                const keys = Object.keys(this.requests), lastUuid = keys[keys.length - 1];

                if (lastUuid !== undefined && this.requestUUID !== lastUuid) {
                    this.requestUUID = lastUuid;
                }
            },

            clearRequests() {
                for (let uuid in this.requests) {
                    if (this.requests.hasOwnProperty(uuid)) {
                        delete this.requests[uuid];
                    }
                }

                this.requestUUID = null;
            },

            deleteAllRequests() {
                this.$api.deleteAllSessionRequests(this.sessionUUID)
                    .then((status) => {
                        if (status.success === true) {
                            this.$izitoast.success({title: 'All requests successfully removed!', zindex: 10});
                        } else {
                            throw new Error(`I've got unsuccessful status`);
                        }
                    })
                    .catch((err) => this.$izitoast.error({
                        title: `Cannot remove all requests: ${err.message}`,
                        zindex: 10
                    }))

                this.clearRequests();
                this.$forceUpdate();
            },

            /**
             * @param {String} uuid
             */
            deleteRequestHandler(uuid) {
                this.$api.deleteSessionRequest(this.sessionUUID, uuid)
                    .then((status) => {
                        if (status.success === true) {
                            this.$izitoast.success({
                                title: `Request with UUID ${uuid} was successfully removed!`,
                                zindex: 10
                            });
                        } else {
                            throw new Error(`I've got unsuccessful status`);
                        }
                    })
                    .catch((err) => this.$izitoast.error({title: `Cannot remove request: ${err.message}`, zindex: 10}))

                // find next request for selection
                const UUIDs = Object.keys(this.requests);
                const currentIndexNum = UUIDs.indexOf(uuid);
                let newCurrentRequestUUID = this.requestUUID; // do not change

                if (uuid !== this.requestUUID) {
                    // do nothing
                } else if (UUIDs[currentIndexNum + 1] !== undefined) {
                    newCurrentRequestUUID = UUIDs[currentIndexNum + 1]; // select next
                } else if (UUIDs[currentIndexNum - 1] !== undefined) {
                    newCurrentRequestUUID = UUIDs[currentIndexNum - 1]; // select previous
                }

                this.requestUUID = newCurrentRequestUUID;

                delete this.requests[uuid];

                this.$forceUpdate();
            },

            /**
             * @param {{statusCode: String|null, contentType: String|null, responseDelay: String|null, responseBody: String|null}} urlSettings
             */
            newUrlHandler(urlSettings) {
                this.$api.startNewSession({
                    content_type: urlSettings.contentType,
                    status_code: urlSettings.statusCode,
                    response_delay: urlSettings.responseDelay,
                    response_body: urlSettings.responseBody,
                })
                    .then((newSessionData) => {
                        newSessionData.uuid
                        this.sessionUUID = newSessionData.uuid;
                        this.$session.setLocalSessionUUID(newSessionData.uuid);

                        this.clearRequests();
                        this.$forceUpdate();
                        this.$izitoast.success({title: 'New session started!', zindex: 10});
                    })
                    .catch((err) => this.$izitoast.error({
                        title: `Cannot create new session: ${err.message}`,
                        zindex: 10
                    }))
            },
        }
    }
</script>

<style scoped>
    .total-requests-count {
        position: relative;
        top: -.15em;
    }

    .request-plate {
        cursor: pointer;
    }

    button .disabled {
        pointer-events: none;
    }
</style>
