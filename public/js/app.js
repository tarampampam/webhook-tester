'use strict';

/** @typedef {Vue} Vue */
/** @typedef {Object} httpVueLoader */

define(
    ['Vue', 'moment', 'axios', 'clipboard', 'izitoast', 'api', 'session', 'vue-loader'],
    (Vue, moment, axios, clipboard, izitoast, api, session) => {
        let isProduction = true,
            clip = new clipboard('.btn');

        // <https://clipboardjs.com/#events>
        clip.on('error', function (e) {
            izitoast.error({title: 'Copying error!', icon: 'fas fa-times'});
        }).on('success', function (e) {
            izitoast.success({title: 'Copied!', message: e.text, icon: 'fas fa-copy', timeout: 3000});
            e.clearSelection();
        });

        if (window.location.hostname.startsWith('127.') || window.location.href.startsWith('file:')) {
            // @link <https://github.com/vuejs/vue-devtools/issues/190#issuecomment-264203810>
            Vue.config.devtools = true; // Disable on "production"
            isProduction = false;
        }

        Vue.prototype.$isProduction = Vue.$isProduction = isProduction;

        // Extending Vue with additional tools/libs
        Vue.prototype.$izitoast = Vue.$izitoast = izitoast;
        Vue.prototype.$clipboard = Vue.$clipboard = clip;
        Vue.prototype.$session = Vue.$session = session;
        Vue.prototype.$moment = Vue.$moment = moment;
        Vue.prototype.$axios = Vue.$axios = axios;
        Vue.prototype.$api = Vue.$api = api;

        new Vue({
            el: '#app',
            template: `<app></app>`,
            components: {
                'app': 'url:vue/app.vue',
            },
        });
    }
);
