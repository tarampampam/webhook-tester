'use strict';

/* global requirejs */

requirejs.config({
    baseUrl: 'js',
    paths: {
        // Promise based HTTP client for the browser and node.js
        // Docs: <https://github.com/axios/axios>
        axios: [
            'https://cdnjs.cloudflare.com/ajax/libs/axios/0.21.1/axios.min',
            'https://cdn.jsdelivr.net/npm/axios@0.21.1/dist/axios.min',
        ],
        // JS framework for UI
        // Docs: <https://vuejs.org/v2/api/>
        Vue: [
            'https://cdnjs.cloudflare.com/ajax/libs/vue/2.6.12/vue.min', // hint: remove `.min` for `vue.js devtools`
            'https://cdn.jsdelivr.net/npm/vue@2.6.12/dist/vue.min',
        ],
        // Router for VueJS
        // Docs: <https://router.vuejs.org/api/>
        VueRouter: [
            'https://cdnjs.cloudflare.com/ajax/libs/vue-router/3.5.1/vue-router.min',
            'https://cdn.jsdelivr.net/npm/vue-router@3.5.1/dist/vue-router.min',
        ],
        // Plugin for runtime loading VueJs components
        // Docs: <https://github.com/FranckFreiburger/http-vue-loader>
        httpVueLoader: [
            'https://cdn.jsdelivr.net/gh/FranckFreiburger/http-vue-loader@1.4.2/src/httpVueLoader.min',
            'https://cdn.jsdelivr.net/npm/http-vue-loader@1.4.2/src/httpVueLoader.min',
        ],
        // Parse, validate, manipulate, and display dates in javascript
        // Docs: <http://momentjs.com/>
        moment: [
            'https://cdnjs.cloudflare.com/ajax/libs/moment.js/2.29.1/moment.min',
            'https://cdn.jsdelivr.net/npm/moment@2.29.1/dist/moment.min',
        ],
        // Modern copy to clipboard
        // Docs: <https://clipboardjs.com>
        clipboard: [
            'https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.8/clipboard.min',
            'https://cdn.jsdelivr.net/npm/clipboard@2.0.8/dist/clipboard.min',
        ],
        // Elegant, responsive, flexible and lightweight notification plugin with no dependencies
        // Docs: <http://izitoast.marcelodolza.com/>
        izitoast: [
            'https://cdnjs.cloudflare.com/ajax/libs/izitoast/1.4.0/js/iziToast.min',
            'https://cdn.jsdelivr.net/npm/izitoast@1.4.0/dist/js/iziToast.min',
        ],
        // Syntax highlighting with language autodetection
        // Docs: <https://highlightjs.org/>
        hljs: [
            'https://cdnjs.cloudflare.com/ajax/libs/highlight.js/10.7.2/highlight.min',
            'https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@10.7.2/build/highlight.min',
        ],
        // WebSocket connection that will automatically reconnect if the connection is dropped
        // Docs: <https://github.com/joewalnes/reconnecting-websocket/>
        reconnectingWebsocket: [
            'https://cdnjs.cloudflare.com/ajax/libs/reconnecting-websocket/1.0.0/reconnecting-websocket.min',
            'https://cdn.jsdelivr.net/gh/joewalnes/reconnecting-websocket@1.0.0/reconnecting-websocket.min',
        ]
    },
    map: {
        '*': {
            // Allow the direct "css!" usage: 'css!/path/to/style'(.css)
            // Docs: <https://github.com/guybedford/require-css>
            css: 'https://cdnjs.cloudflare.com/ajax/libs/require-css/0.1.10/css.min.js',
        }
    },
    shim: {
        Vue: {
            exports: 'Vue'
        },
        izitoast: {
            deps: ['css!https://cdnjs.cloudflare.com/ajax/libs/izitoast/1.4.0/css/iziToast.min.css']
        },
        hljs: {
            deps: ['css!https://cdnjs.cloudflare.com/ajax/libs/highlight.js/10.7.2/styles/obsidian.min.css']
        },
    },
});

// Start loading the main app file.
requirejs(['app']);
