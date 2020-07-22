'use strict';

requirejs.config({
    baseUrl: 'js',
    paths: {
        // Promise based HTTP client for the browser and node.js
        // Docs: <https://github.com/axios/axios>
        axios: [
            'https://cdnjs.cloudflare.com/ajax/libs/axios/0.19.2/axios.min',
            'https://cdn.jsdelivr.net/npm/axios@0.19.2/dist/axios.min',
        ],
        // JS framework for UI
        // Docs: <https://vuejs.org/v2/api/>
        Vue: [
            'https://cdnjs.cloudflare.com/ajax/libs/vue/2.6.11/vue.min', // remove `.min` for using with `vue.js devtools`
            'https://cdn.jsdelivr.net/npm/vue@2.6.11/dist/vue.min',
        ],
        // Router for VueJS
        // Docs: <https://router.vuejs.org/api/>
        VueRouter: [
            'https://cdnjs.cloudflare.com/ajax/libs/vue-router/3.2.0/vue-router.min',
            'https://cdn.jsdelivr.net/npm/vue-router@3.2.0/dist/vue-router.min',
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
            'https://cdnjs.cloudflare.com/ajax/libs/moment.js/2.27.0/moment.min',
            'https://cdn.jsdelivr.net/npm/moment@2.27.0/dist/moment.min',
        ],
        // Modern copy to clipboard. No Flash. Just 2kb
        // Docs: <https://clipboardjs.com>
        clipboard: [
            'https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.6/clipboard.min',
            'https://cdn.jsdelivr.net/npm/clipboard@2.0.6/dist/clipboard.min',
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
            'https://cdnjs.cloudflare.com/ajax/libs/highlight.js/10.1.1/highlight.min',
            'https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@10.1.1/build/highlight.min'
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
            deps: ['css!https://cdnjs.cloudflare.com/ajax/libs/highlight.js/10.1.1/styles/obsidian.min.css']
        },
    },
});

// Start loading the main app file.
requirejs(['app']);
