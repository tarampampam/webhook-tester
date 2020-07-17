'use strict';

requirejs.config({
    baseUrl: 'js',
    paths: {
        // Promise based HTTP client for the browser and node.js
        // Docs: <https://github.com/axios/axios>
        axios: 'https://cdnjs.cloudflare.com/ajax/libs/axios/0.19.2/axios.min',
        // JS framework for UI
        // Docs: <https://vuejs.org/v2/api/>
        Vue: 'https://cdnjs.cloudflare.com/ajax/libs/vue/2.6.11/vue.min',
        // Plugin for runtime loading VueJs components
        // Docs: <https://github.com/FranckFreiburger/http-vue-loader>
        httpVueLoader: 'https://cdn.jsdelivr.net/gh/FranckFreiburger/http-vue-loader@1.4.2/src/httpVueLoader.min',
    },
    shim: {
        Vue: {
            exports: 'Vue'
        }
    },
});

// Start loading the main app file.
requirejs(['app']);
