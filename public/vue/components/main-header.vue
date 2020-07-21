<template>
    <header class="navbar navbar-expand flex-column flex-md-row flex-sm-row navbar-dark bg-primary">
        <span class="navbar-brand mr-0 mr-md-2">
            WebHook Tester
        </span>

        <div class="mr-auto">
            <ul class="navbar-nav flex-row d-none d-sm-block">
                <li class="nav-item" data-toggle="modal" data-target="#help-modal">
                    <span class="nav-link"><i class="fas fa-question mr-1"></i> Help</span>
                </li>
            </ul>
        </div>

        <div class="modal fade" id="help-modal" tabindex="-1" role="dialog" aria-hidden="true">
            <div class="modal-dialog modal-lg modal-dialog-centered" role="document">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">What is WebHook Tester?</h5>
                        <button type="button" class="close" data-dismiss="modal" aria-label="Close" title="Close">
                            <span aria-hidden="true">&times;</span>
                        </button>
                    </div>
                    <div class="modal-body">
                        <p>
                            <strong>Webhook Tester</strong> allows you to easily test webhooks and other types of HTTP
                            requests. Here's your unique URL:
                        <p>
                            <code>{{ currentWebHookUrl }}</code>
                            <button class="btn btn-outline-info btn-sm ml-2" v-bind:data-clipboard-text="currentWebHookUrl">
                                <i class="fas fa-copy mr-1"></i> Copy
                            </button>
                            <a target="_blank"
                               class="btn btn-outline-info btn-sm"
                               :href="currentWebHookUrl">
                                <i class="fas fa-external-link-alt pr-1"></i> Try it!
                            </a>
                        </p>
                        <p>Any requests sent to that URL are instantly logged here - you don't even have to refresh.</p>
                        <hr/>
                        <p>Append a status code to the url, e.g.:</p>
                        <p>
                            <code>{{ currentWebHookUrl }}/404</code>
                            <button class="btn btn-outline-info btn-sm ml-2"
                                    v-bind:data-clipboard-text="currentWebHookUrl + '/404'">
                                <i class="fas fa-copy mr-1"></i> Copy
                            </button>
                            <a target="_blank"
                               class="btn btn-outline-info btn-sm"
                               :href="currentWebHookUrl + '/404'">
                                <i class="fas fa-external-link-alt pr-1"></i> Try it!
                            </a>
                        </p>
                        <p>So the URL will respond with a <code>404: Not Found</code>.</p>
                        <p>
                            You can bookmark this page to go back to the request contents at any time. Requests and the
                            tokens for the URL expire <strong>after {{ sessionLifetimeDays }}</strong> days of not being
                            used.
                        </p>
                        <hr />
                        <p class="small">
                            Current application version: <strong> v{{ version }}</strong>
                        </p>
                    </div>
                </div>
            </div>
        </div>

        <div class="form-inline my-2 my-lg-0">
            <button class="btn btn-success my-2 my-sm-0 border-0" v-bind:data-clipboard-text="currentWebHookUrl">
                <i class="fas fa-copy mr-1"></i> Copy URL
            </button>
            <button class="btn btn-info my-2 ml-2 my-sm-0 border-0" data-toggle="modal" data-target="#new-url-modal">
                <i class="fas fa-plus mr-1"></i> New URL
            </button>

            <div class="modal fade" id="new-url-modal" tabindex="-1" role="dialog" aria-hidden="true">
                <div class="modal-dialog modal-dialog-centered" role="document">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">Configure URL</h5>
                            <button type="button" class="close" data-dismiss="modal" aria-label="Close">
                                <span aria-hidden="true">&times;</span>
                            </button>
                        </div>
                        <div class="modal-body">
                            <p>
                                You have the ability to customize how your URL will respond by changing the status
                                code, content-type header and the content.
                            </p>
                            <div class="form-group row">
                                <label class="d-block col-md-4 col-form-label col-form-label-sm text-right"
                                       for="default-status-code">Default status code</label>
                                <div class="col-md-8">
                                    <input type="number"
                                           autocomplete="off"
                                           min="100"
                                           max="530"
                                           class="form-control w-100"
                                           id="default-status-code"
                                           placeholder="200"
                                           v-model="newUrlData.statusCode">
                                </div>
                            </div>
                            <div class="form-group row">
                                <label class="d-block col-md-4 col-form-label col-form-label-sm text-right"
                                       for="content-type">Content Type</label>
                                <div class="col-md-8">
                                    <input type="text"
                                           autocomplete="off"
                                           minlength="1"
                                           maxlength="32"
                                           class="form-control w-100"
                                           id="content-type"
                                           placeholder="text/plain"
                                           v-model="newUrlData.contentType">
                                </div>
                            </div>
                            <div class="form-group row">
                                <label class="d-block col-md-4 col-form-label col-form-label-sm text-right"
                                       for="response-delay">Delay before response</label>
                                <div class="col-md-8">
                                    <input type="number"
                                           autocomplete="off"
                                           min="0"
                                           max="30"
                                           class="form-control w-100"
                                           id="response-delay"
                                           placeholder="0"
                                           v-model="newUrlData.responseDelay">
                                </div>
                            </div>
                            <div class="form-group row">
                                <label class="d-block col-md-4 col-form-label col-form-label-sm text-right"
                                       for="response-body">Response body</label>
                                <div class="col-md-8">
                                    <textarea autocomplete="off"
                                              class="form-control w-100"
                                              id="response-body"
                                              rows="3"
                                              maxlength="2048"
                                              placeholder=""
                                              v-model="newUrlData.responseBody"></textarea>
                                </div>
                            </div>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-dismiss="modal">Cancel</button>
                            <button type="button" class="btn btn-primary" data-dismiss="modal" @click="newURL">Create</button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </header>
</template>

<script>
    /* global module */

    'use strict';

    module.exports = {
        props: {
            currentWebHookUrl: {
                type: String,
                default: 'URL was not defined',
            },
            sessionLifetimeSec: {
                type: Number,
                default: null,
            },
            version: {
                type: String,
                default: 'unknown',
            },
        },

        data: function () {
            return {
                newUrlData: {
                    statusCode: null,
                    contentType: null,
                    responseDelay: null,
                    responseBody: null,
                },
            }
        },

        computed: {
            /**
             * @returns {Number}
             */
            sessionLifetimeDays: function () {
                if (typeof this.sessionLifetimeSec === 'number') {
                    return Number((this.sessionLifetimeSec / 24 / 60 / 60).toFixed(1));
                }

                return 0;
            },
        },

        methods: {
            newURL() {
                // <https://michaelnthiessen.com/pass-function-as-prop/>
                this.$emit('on-new-url', {
                    statusCode: this.newUrlData.statusCode,
                    contentType: this.newUrlData.contentType,
                    responseDelay: this.newUrlData.responseDelay,
                    responseBody: this.newUrlData.responseBody,
                });
            },
        }
    }
</script>

<style scoped>
    .nav-link {
        cursor: pointer;
    }
</style>
