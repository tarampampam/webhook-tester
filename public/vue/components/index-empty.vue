<template>
    <div>
        <h4 class="mt-2">
            WebHook Tester allows you to easily test webhooks and other types of HTTP requests
        </h4>
        <p class="text-muted">
            Any requests sent to that URL are logged here instantly — you don't even have to refresh!
        </p>
        <hr/>
        <p>Here's your unique URL that was created just now:</p>
        <p>
            <code id="current-webhook-url-text">{{ currentWebHookUrl }}</code>
            <button class="btn btn-primary btn-sm ml-2"
                    data-clipboard-target="#current-webhook-url-text">
                <i class="fas fa-copy mr-1"></i> Copy
            </button>
            <a target="_blank"
               class="btn btn-primary btn-sm"
               :href="currentWebHookUrl">
                <i class="fas fa-external-link-alt pr-1"></i> Open in a new tab
            </a>
            <button class="btn btn-primary btn-sm" @click="testXHR" title="Using random HTTP method">
                <i class="fas fa-running mr-1"></i> XHR
            </button>
        </p>
        <p>
            Send simple POST request (execute next command in your terminal without leaving this page):
        <p>
            <code>
                $ <span id="current-webhook-curl-text">curl -v -X POST -d "foo=bar" {{ currentWebHookUrl }}</span>
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
        },

        data: function () {
            return {
                xhrMethods: ['post', 'put', 'delete', 'patch'],
            }
        },

        methods: {
            testXHR() {
                const payload = {
                    xhr: 'test',
                    now: Math.floor(Date.now() / 1000),
                };

                this.$axios({
                    method: this.xhrMethods[Math.floor(Math.random() * this.xhrMethods.length)],
                    url: this.currentWebHookUrl,
                    data: payload
                })
                    .catch((err) => this.$izitoast.error({title: err.message}));

                this.$izitoast.success({title: 'Background request was sent', timeout: 2000});
            },
        }
    }
</script>

<style scoped>
</style>
