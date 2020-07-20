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

        <div class="modal fade" id="help-modal" tabindex="-1" role="dialog" aria-labelledby="exampleModalCenterTitle"
             aria-hidden="true">
            <div class="modal-dialog modal-lg modal-dialog-centered" role="document">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="exampleModalLongTitle">What is WebHook Tester?</h5>
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
                            <a :href="currentWebHookUrl" target="_blank">(try it!)</a>
                        </p>
                        <p>Any requests sent to that URL are instantly logged here - you don't even have to refresh.</p>
                        <hr/>
                        <p>Append a status code to the url, e.g.:</p>
                        <p><code>{{ currentWebHookUrl }}/404</code></p>
                        <p>So the URL will respond with a <code>404: Not Found</code>.</p>
                        <p>
                            You can bookmark this page to go back to the request contents at any time. Requests and the
                            tokens for the URL expire <strong>after {{ requestsLifetime }}</strong> days of not being
                            used.
                        </p>
                    </div>
                </div>
            </div>
        </div>

        <div class="form-inline my-2 my-lg-0">
            <button class="btn btn-success my-2 my-sm-0 border-0" v-bind:data-clipboard-text="currentWebHookUrl">
                <i class="fas fa-copy mr-1"></i> Copy URL
            </button>
            <button class="btn btn-info my-2 ml-2 my-sm-0 border-0" @click="newURL">
                <i class="fas fa-plus mr-1"></i> New URL
            </button>
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
            requestsLifetime: {
                type: Number,
                default: NaN,
            },
        },

        methods: {
            newURL() {
                // <https://michaelnthiessen.com/pass-function-as-prop/>
                this.$emit('on-new-url');
            },
        }
    }
</script>

<style scoped>
    .nav-link {
        cursor: pointer;
    }
</style>
