<template>
    <div>
        <main-header
            :current-web-hook-url="sessionRequestURI"
            @on-new-url="newUrlHandler"
        ></main-header>

        <div class="container-fluid">
            <div class="row flex-xl-nowrap">
                <div class="sidebar col-sm-5 col-md-4 col-xl-3 px-2 py-0">
                    <div class="pl-3 pt-4 pr-3 pb-3">
                        <div class="d-flex w-100 justify-content-between">
                            <h5 class="text-uppercase mb-0">Requests <span
                                class="badge badge-primary badge-pill total-requests-count"
                            >{{ getRequestsCount() }}</span></h5>
                            <button type="button"
                                    class="btn btn-outline-danger btn-sm"
                                    v-if="getRequestsCount()"
                                    @click="deleteAllRequests()">Delete all
                            </button>
                        </div>
                    </div>

                    <div class="list-group" v-if="getRequestsCount()">
                        <request-plate
                            v-for="(request, uuid) in this.requests"
                            class="request-plate"
                            :uuid="uuid"
                            :ip="request.ip"
                            :method="request.method"
                            :when="request.when"
                            @on-delete="deleteRequestHandler"
                        ></request-plate>
                    </div>
                    <div v-else class="text-muted text-center mt-3">
                        <span class="spinner-border spinner-border-sm mr-1"></span> Waiting for first request
                    </div>
                </div>

                <main class="col-sm-7 col-md-8 col-xl-9 py-3 pl-md-4" role="main">
                    <div v-if="getRequestsCount()">
                        <div class="row pt-2">
                            <requests-navigation class="col-6"></requests-navigation>
                            <settings class="col-6 text-right"></settings>
                        </div>

                        <request-details class="pt-3"></request-details>
                        <request-body></request-body>
                    </div>
                    <div v-else>
                        <h4 class="mt-2">
                            WebHook Tester allows you to easily test webhooks and other types of HTTP requests
                        </h4>
                        <p class="text-muted">
                            Any requests sent to that URL are logged here instantly — you don't even have to refresh!
                        </p>
                        <hr/>
                        <p>Here's your unique URL that was created just now:</p>
                        <p>
                            <code id="current-webhook-url-text">{{ sessionRequestURI }}</code>
                            <button class="btn btn-primary btn-sm ml-2"
                                    data-clipboard-target="#current-webhook-url-text">
                                <i class="fas fa-copy mr-1"></i> Copy
                            </button>
                            <a target="_blank"
                               class="btn btn-primary btn-sm"
                               :href="sessionRequestURI">
                                <i class="fas fa-external-link-alt pr-1"></i> Open in a new tab
                            </a>
                        </p>
                        <p>
                            Send simple POST request (execute next command in your terminal without leaving this page):
                        <p>
                            <code>
                                $ <span id="current-webhook-curl-text">curl -v -X POST -d "foo=bar" {{ sessionRequestURI }}</span>
                            </code>
                            <button class="btn btn-primary btn-sm ml-2"
                                    data-clipboard-target="#current-webhook-curl-text">
                                <i class="fas fa-copy mr-1"></i> Copy
                            </button>
                        </p>
                        <hr/>
                        <p>
                            Bookmark this page to go back to the requests at any time. For more info, click
                            <strong>Help</strong>.
                        </p>
                        <p>
                            Click <strong>New URL</strong> to create a new url with the ability to customize status
                            code, response body, etc.
                        </p>
                    </div>
                </main>
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
            'requests-navigation': 'url:/vue/components/requests-navigation.vue',
            'request-details': 'url:/vue/components/request-details.vue',
            'request-body': 'url:/vue/components/request-body.vue',
            'settings': 'url:/vue/components/settings.vue',
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

                session: {
                    UUID: null,
                },
            }
        },

        mounted() {
            document.getElementById('main-loader').remove();

            this.initSession();
        },

        computed: {
            /**
             * @returns {String}
             */
            sessionRequestURI: function () {
                let uuid = this.session.UUID === null
                    ? '________-____-____-____-____________'
                    : this.session.UUID;

                return `${window.location.origin}/${uuid}`;
            }
        },

        methods: {
            initSession() {
                const localSessionUUID = this.$session.getLocalSessionUUID();

                /**
                 * @param {Object.<string, APIRecordedRequest>} requests
                 */
                const initSessionRequests = (requests) => {
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
                            };
                        }
                    }
                };

                const startNewSession = () => {
                    this.$api.startNewSession()
                        .then((newSessionData) => {
                            this.session.UUID = newSessionData.uuid;
                            this.$session.setLocalSessionUUID(newSessionData.uuid);

                            this.$api.getAllSessionRequests(newSessionData.uuid)
                                .then((requests) => {
                                    initSessionRequests(requests);
                                    this.$forceUpdate();
                                })
                                .catch((err) => this.$izitoast.error({title: `Cannot retrieve requests: ${err.message}`}))
                        })
                        .catch((err) => this.$izitoast.error({title: `Cannot create new session: ${err.message}`}))
                };

                if (localSessionUUID !== null) {
                    this.session.UUID = localSessionUUID;

                    this.$api.getAllSessionRequests(localSessionUUID)
                        .then((requests) => {
                            initSessionRequests(requests);
                            this.$forceUpdate();
                        })
                        .catch(() => startNewSession())
                } else {
                    startNewSession()
                }
            },

            /**
             * @returns {Number}
             */
            getRequestsCount() {
                return Object.keys(this.requests).length;
            },

            clearRequests() {
                for (let uuid in this.requests) {
                    if (this.requests.hasOwnProperty(uuid)) {
                        delete this.requests[uuid];
                    }
                }
            },

            deleteAllRequests() {
                this.$api.deleteAllSessionRequests(this.session.UUID)
                    .then((status) => {
                        if (status.success === true) {
                            this.$izitoast.success({title: 'All requests successfully removed!'});
                        } else {
                            throw new Error(`I've got unsuccessful status`);
                        }
                    })
                    .catch((err) => this.$izitoast.error({title: `Cannot remove all requests: ${err.message}`}))

                this.clearRequests();
                this.$forceUpdate();
            },

            /**
             * @param {String} uuid
             */
            deleteRequestHandler(uuid) {
                this.$api.deleteSessionRequest(this.session.UUID, uuid)
                    .then((status) => {
                        if (status.success === true) {
                            this.$izitoast.success({title: 'Request successfully removed!'});
                        } else {
                            throw new Error(`I've got unsuccessful status`);
                        }
                    })
                    .catch((err) => this.$izitoast.error({title: `Cannot remove request: ${err.message}`}))

                delete this.requests[uuid];

                this.$forceUpdate();
            },

            newUrlHandler() {
                this.$api.startNewSession()
                    .then((newSessionData) => {
                        newSessionData.uuid
                        this.session.UUID = newSessionData.uuid;
                        this.$session.setLocalSessionUUID(newSessionData.uuid);

                        this.clearRequests();
                        this.$forceUpdate();
                        this.$izitoast.success({title: 'New session started!'});
                    })
                    .catch((err) => this.$izitoast.error({title: `Cannot create new session: ${err.message}`}))
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

    @media (min-width: 992px) {
        .sidebar {
            max-width: 315px;
        }
    }
</style>
