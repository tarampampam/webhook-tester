<template>
  <div>
    <h4 class="mt-2">
      WebHook Tester allows you to easily test webhooks and other types of HTTP requests
    </h4>
    <p class="text-muted">
      Any requests sent to that URL are logged here instantly — you don't even have to refresh!
    </p>
    <hr>
    <p>
      Here's your unique URL that was created just now:
    </p>
    <p>
      <code id="current-webhook-url-text">{{ currentWebHookUrl }}</code>
      <button
        class="btn btn-primary btn-sm ms-2"
        data-clipboard-target="#current-webhook-url-text"
        data-clipboard
      >
        <font-awesome-icon
          icon="fa-regular fa-copy"
          class="pe-1"
        />
        Copy
      </button>
      <a
        target="_blank"
        class="btn btn-primary btn-sm ms-1"
        :href="currentWebHookUrl"
      >
        <font-awesome-icon
          icon="fa-arrow-up-right-from-square"
          class="pe-1"
        />
        Open in a new tab
      </a>
      <button
        class="btn btn-primary btn-sm ms-1"
        @click="testXHR"
        title="Using random HTTP method"
      >
        <font-awesome-icon
          icon="fa-solid fa-person-running"
          class="pe-1"
        />
        XHR
      </button>
    </p>
    <p>
      Send simple POST request (execute next command in your terminal without leaving this page):
    </p>
    <p>
      <code>
        $ <span id="current-webhook-curl-text">curl -v -X POST -d "foo=bar" {{ currentWebHookUrl }}</span>
      </code>
      <button
        class="btn btn-primary btn-sm ms-2"
        data-clipboard-target="#current-webhook-curl-text"
        data-clipboard
      >
        <font-awesome-icon
          icon="fa-regular fa-copy"
          class="me-1"
        />
        Copy
      </button>
    </p>
    <hr>
    <p>
      Bookmark this page to go back to the requests at any time. For more info, click <strong>Help</strong>.
    </p>
    <p>
      Click <strong>New URL</strong> to create a new url with the ability to customize status
      code, response body, etc.
    </p>
  </div>
</template>

<script lang="ts">
import {defineComponent} from 'vue'
import iziToast from 'izitoast'
import {FontAwesomeIcon} from '@fortawesome/vue-fontawesome'

const xhrMethods = ['post', 'put', 'delete', 'patch']

export default defineComponent({
  components: {
    'font-awesome-icon': FontAwesomeIcon,
  },
  props: {
    currentWebHookUrl: {
      type: String,
      default: 'URL was not defined',
    },
  },

  methods: {
    testXHR() {
      const payload = {
        xhr: 'test',
        now: Math.floor(Date.now() / 1000),
      }

      fetch(new Request(this.currentWebHookUrl, { // TODO use API client function for this
        method: xhrMethods[Math.floor(Math.random() * xhrMethods.length)].toUpperCase(),
        body: JSON.stringify(payload),
      }))
        .catch((err) => iziToast.error({title: err.message}));

      iziToast.success({title: 'Background request was sent', timeout: 2000});
    },
  }
})
</script>

<style lang="scss" scoped>
hr {
  opacity: .05;
}
</style>
