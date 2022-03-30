'use strict';

/* global requirejs */

requirejs.config({
    baseUrl: 'js',
    paths: {
        // Promise based HTTP client for the browser and node.js
        // Docs: <https://github.com/axios/axios>
        axios: [
            '/assets/js/axios.min',
        ],
        // Yet another Base64 transcoder
        // Docs: <https://github.com/dankogai/js-base64>
        Base64: [
            '/assets/js/base64.min'
        ],
        // JS framework for UI
        // Docs: <https://vuejs.org/v2/api/>
        Vue: [
            '/assets/js/vue.min', // hint: remove `.min` for `vue.js devtools`
        ],
        // Router for VueJS
        // Docs: <https://router.vuejs.org/api/>
        VueRouter: [
            '/assets/js/vue-router.min',
        ],
        // Plugin for runtime loading VueJs components
        // Docs: <https://github.com/FranckFreiburger/http-vue-loader>
        httpVueLoader: [
            '/assets/js/httpVueLoader.min',
        ],
        // Parse, validate, manipulate, and display dates in javascript
        // Docs: <http://momentjs.com/>
        moment: [
            '/assets/js/moment.min',
        ],
        // Modern copy to clipboard
        // Docs: <https://clipboardjs.com>
        clipboard: [
            '/assets/js/clipboard.min',
        ],
        // Elegant, responsive, flexible and lightweight notification plugin with no dependencies
        // Docs: <http://izitoast.marcelodolza.com/>
        izitoast: [
            '/assets/js/iziToast.min',
        ],
        // Syntax highlighting with language autodetection
        // Docs: <https://highlightjs.org/>
        hljs: [
            '/assets/js/highlight.min',
        ],
        // WebSocket connection that will automatically reconnect if the connection is dropped
        // Docs: <https://github.com/joewalnes/reconnecting-websocket/>
        reconnectingWebsocket: [
            '/assets/js/reconnecting-websocket.min',
        ]
    },
    map: {
        '*': {
            // Allow the direct "css!" usage: 'css!/path/to/style'(.css)
            // Docs: <https://github.com/guybedford/require-css>
            css: '/assets/js/require-css.min.js',
        }
    },
    shim: {
        Vue: {
            exports: 'Vue'
        },
        izitoast: {
            deps: ['css!/assets/css/iziToast.min.css']
        },
        hljs: {
            deps: ['css!/assets/css/obsidian.min.css']
        },
    },
});

// Start loading the main app file.
requirejs(['app']);
