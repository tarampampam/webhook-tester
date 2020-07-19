'use strict';

requirejs.config({
    baseUrl: 'js',
    paths: {
        // Promise based HTTP client for the browser and node.js
        // Docs: <https://github.com/axios/axios>
        axios: 'https://cdnjs.cloudflare.com/ajax/libs/axios/0.19.2/axios.min',
        // JS framework for UI
        // Docs: <https://vuejs.org/v2/api/>
        Vue: 'https://cdnjs.cloudflare.com/ajax/libs/vue/2.6.11/vue',
        // Vue: 'https://cdnjs.cloudflare.com/ajax/libs/vue/2.6.11/vue.min', // @todo: ENABLE FOR PRODUCTION USAGE
        // Router for VueJS
        // Docs: <https://router.vuejs.org/api/>
        VueRouter: 'https://cdnjs.cloudflare.com/ajax/libs/vue-router/3.2.0/vue-router.min',
        // Plugin for runtime loading VueJs components
        // Docs: <https://github.com/FranckFreiburger/http-vue-loader>
        httpVueLoader: 'https://cdn.jsdelivr.net/gh/FranckFreiburger/http-vue-loader@1.4.2/src/httpVueLoader.min',
        // Parse, validate, manipulate, and display dates in javascript
        // Docs: <http://momentjs.com/>
        moment: 'https://cdnjs.cloudflare.com/ajax/libs/moment.js/2.27.0/moment.min',
        // Modern copy to clipboard. No Flash. Just 2kb
        // Docs: <https://clipboardjs.com>
        clipboard: 'https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.6/clipboard.min',
        // Elegant, responsive, flexible and lightweight notification plugin with no dependencies
        // Docs: <http://izitoast.marcelodolza.com/>
        izitoast: 'https://cdnjs.cloudflare.com/ajax/libs/izitoast/1.4.0/js/iziToast.min',
    },
    map: {
        '*': {
            // Allow the direct "css!" usage: 'css!/path/to/style'(.css)
            // Docs: <https://github.com/guybedford/require-css>
            css: 'https://cdnjs.cloudflare.com/ajax/libs/require-css/0.1.10/css.min.js'
        }
    },
    shim: {
        Vue: {
            exports: 'Vue'
        },
        izitoast: {
            deps: ['css!https://cdnjs.cloudflare.com/ajax/libs/izitoast/1.4.0/css/iziToast.min.css']
        },
    },
});

// Start loading the main app file.
requirejs(['app']);
