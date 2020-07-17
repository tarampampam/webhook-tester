'use strict';

/** @typedef {Vue} Vue */
/** @typedef {Object} httpVueLoader */

define(['Vue', 'moment', 'axios', 'vue-loader'], (Vue, moment, axios) => {
    let isProduction = true;

    if (window.location.hostname.startsWith('127.') || window.location.href.startsWith('file:')) {
        // @link <https://github.com/vuejs/vue-devtools/issues/190#issuecomment-264203810>
        Vue.config.devtools = true; // Disable on "production"
        isProduction = false;
    }

    Vue.prototype.$isProduction = Vue.$isProduction = isProduction;

    // Extending Vue with additional tools/libs
    Vue.prototype.$moment = Vue.$moment = moment;
    Vue.prototype.$axios = Vue.$axios = axios;

    new Vue({
        el: '#app',
        template: `<app></app>`,
        components: {
            'app': 'url:vue/app.vue',
        },
    });
});
