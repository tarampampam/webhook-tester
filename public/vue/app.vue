<template>
    <div>
        <main-header></main-header>

        <div class="container-fluid">
            <div class="row flex-xl-nowrap">
                <div class="col-sm-5 col-md-4 col-xl-3 px-2 py-0">
                    <div class="pl-3 pt-4 pr-3 pb-3">
                        <div class="d-flex w-100 justify-content-between">
                            <h5 class="text-uppercase mb-0">Requests <span
                                class="badge badge-primary badge-pill total-requests-count"
                            >{{ getRequestsCount() }}</span></h5>
                            <button type="button"
                                    class="btn btn-outline-danger btn-sm"
                                    v-if="getRequestsCount()"
                            >Delete all</button>
                        </div>
                    </div>

                    <div class="list-group" v-if="getRequestsCount()">
                        <request-plate
                            v-for="(request, uuid) in this.requests"
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
                    <div class="row pt-2">
                        <requests-navigation class="col-6"></requests-navigation>
                        <settings class="col-6 text-right"></settings>
                    </div>

                    <request-details class="pt-3"></request-details>
                    <request-body></request-body>
                </main>
            </div>
        </div>
    </div>
</template>

<script>
    /* global module */

    'use strict';

    module.exports = {
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
                    'cd5e695f-1784-4dcf-9b3f-ef66c9a0aaaa': {
                        ip: '1.1.1.1',
                        method: 'post',
                        when: pastDate,
                    },
                    '69b9131e-1594-4836-af86-f2529fb7bbbb': {
                        ip: '2.2.2.2',
                        method: 'get',
                        when: new Date,
                    },
                },
            }
        },

        mounted() {
            document.getElementById('main-loader').remove();
        },

        methods: {
            getRequestsCount() {
                return Object.keys(this.requests).length;
            },

            deleteRequestHandler(uuid) {
                console.warn(`Removing request with UUID ${uuid}`);

                delete this.requests[uuid];
                // @TODO: Send request to the API for request deletion

                this.$forceUpdate();
            }
        }
    }
</script>

<style scoped>
    .total-requests-count {
        position: relative;
        top: -.15em;
    }
</style>
