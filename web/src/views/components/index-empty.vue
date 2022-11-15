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
        $ <span id="current-webhook-curl-text">curl -v -X POST --data '{"foo": "bar"}' {{ currentWebHookUrl }}</span>
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

    <p>
      Code examples in different languages:
    </p>

    <ul class="nav nav-pills">
      <li
        v-for="btn in [
          {lang: 'javascript', icon: 'fa-brands fa-js', title: 'JavaScript'},
          {lang: 'node', icon: 'fa-brands fa-node-js', title: 'Node.js'},
          {lang: 'go', icon: 'fa-brands fa-golang', title: 'Go'},
          {lang: 'java', icon: 'fa-brands fa-java', title: 'Java'},
          {lang: 'python', icon: 'fa-brands fa-python', title: 'Python'},
          {lang: 'php', icon: 'fa-brands fa-php', title: 'PHP'},
        ]"
        :key="btn.lang"
      >
        <span
          class="btn nav-link ps-4 pe-4 pt-1 pb-1"
          :class="{ 'active': snippetLang === btn.lang }"
          @click="snippetLang=btn.lang"
        >
          <font-awesome-icon
            :icon="btn.icon"
            class="pe-1"
          /> {{ btn.title }}
        </span>
      </li>
    </ul>
    <div class="tab-content pt-2 pb-2">
      <div
        class="tab-pane active"
      >
        <highlightjs
          class="highlightjs"
          :code="snippetCode"
          autodetect
        />
      </div>
    </div>

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
import hljsVuePlugin from '@highlightjs/vue-plugin'

const xhrMethods = ['post', 'put', 'delete', 'patch']

type SnippetLang = 'javascript' | 'node' | 'java' | 'python' | 'php' | 'go'

export default defineComponent({
  components: {
    'highlightjs': hljsVuePlugin.component,
    'font-awesome-icon': FontAwesomeIcon,
  },

  props: {
    currentWebHookUrl: {
      type: String,
      default: 'URL was not defined',
    },
  },

  data(): {
    snippetLang: SnippetLang
  } {
    return {
      snippetLang: 'javascript'
    }
  },

  computed: {
    snippetCode(): string {
      switch (this.snippetLang) {
        case 'javascript':
          return `const options = {method: 'GET'};

fetch('${this.currentWebHookUrl}', options)
\t.then(response => response.json())
\t.then(response => console.log(response))
\t.catch(err => console.error(err));`

        case 'node':
          return `const fetch = require('node-fetch');

fetch('${this.currentWebHookUrl}', {method: 'GET'})
\t.then(res => res.json())
\t.then(json => console.log(json))
\t.catch(err => console.error('error:' + err));`

        case 'java':
          return `HttpRequest request = HttpRequest.newBuilder()
\t.uri(URI.create("${this.currentWebHookUrl}"))
\t.method("GET", HttpRequest.BodyPublishers.noBody())
\t.build();

HttpResponse<String> response = HttpClient.newHttpClient().send(request, HttpResponse.BodyHandlers.ofString());
System.out.println(response.body());`

        case 'python':
          return `import requests

response = requests.request("GET", "${this.currentWebHookUrl}", data="")

print(response.text)`

        case 'php':
          return `<?php

$client = new GuzzleHttp\\Client(); // https://docs.guzzlephp.org/en/stable/
$response = $client->request('GET', '${this.currentWebHookUrl}');

echo $response->getBody();`

        case 'go':
          return `package main

import (
\t"fmt"
\t"net/http"
\t"io/ioutil"
)

func main() {
\treq, _ := http.NewRequest(http.MethodGet, "${this.currentWebHookUrl}", http.NoBody)
\tres, _ := http.DefaultClient.Do(req) // handle the error

\tdefer res.Body.Close()

\tbody, _ := ioutil.ReadAll(res.Body) // handle the error

\tfmt.Println(string(body))
}`
      }

      return ''
    },
  },

  methods: {
    testXHR() {
      const payload = {
        xhr: 'test',
        now: Math.floor(Date.now() / 1000),
      }

      fetch(new Request(this.currentWebHookUrl, {
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

.highlightjs {
  tab-size: 2;
  margin-bottom: 0;
  word-wrap: break-word;
  white-space: pre-wrap;
}
</style>
