'use strict';

/** @typedef {Vue} Vue */
/** @typedef {Object} httpVueLoader */

define(
    ['Vue', 'VueRouter', 'moment', 'axios', 'clipboard', 'izitoast', 'api', 'session', 'pusher', 'highlightjs', 'vue-loader'],
    (Vue, VueRouter, moment, axios, clipboard, izitoast, api, session, pusher) => {
        let isProduction = true;

        const clip = new clipboard('.btn');

        // <https://clipboardjs.com/#events>
        clip.on('error', function (e) {
            izitoast.error({title: 'Copying error!', icon: 'fas fa-times'});
        }).on('success', function (e) {
            izitoast.success({title: 'Copied!', message: e.text, icon: 'fas fa-copy', timeout: 4000});
            e.clearSelection();
        });

        if (window.location.hostname.startsWith('127.') || window.location.href.startsWith('file:')) {
            // @link <https://github.com/vuejs/vue-devtools/issues/190#issuecomment-264203810>
            Vue.config.devtools = true;
            pusher.logToConsole = true;
            isProduction = false;
        }

        try {
            window.localStorage.getItem('test');
        } catch (e) {
            izitoast.error({
                title: 'Local storage not accessible!',
                message: 'Please, allow this site to use browser local storage',
                icon: 'fas fa-times',
                close: false,
                progressBar: false,
                drag: false,
                timeout: 0,
                position: 'center',
                overlay: true,
            });
        }

        const router = new VueRouter({
            mode: 'hash',
            routes: [{path: `/`, name: 'index'}, {path: `/:sessionUUID?/:requestUUID?`, name: 'request', props: true}],
        });

        Vue.use(VueRouter);

        Vue.prototype.$isProduction = Vue.$isProduction = isProduction;

        // Extending Vue with additional tools/libs
        Vue.prototype.$izitoast = Vue.$izitoast = izitoast;
        Vue.prototype.$clipboard = Vue.$clipboard = clip;
        Vue.prototype.$session = Vue.$session = session;
        Vue.prototype.$pusher = Vue.$pusher = pusher;
        Vue.prototype.$moment = Vue.$moment = moment;
        Vue.prototype.$axios = Vue.$axios = axios;
        Vue.prototype.$api = Vue.$api = api;

        return new Vue({
            router,
            el: '#app',
            template: `<app></app>`,
            components: {
                'app': 'url:vue/app.vue',
            },
        });
    }
);
