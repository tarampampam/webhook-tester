'use strict';

define(['Vue', 'vue-loader', 'VueRouter'], (Vue, vueLoader, VueRouter) => {
    const router =  new VueRouter({
        mode: 'hash',
        linkActiveClass: 'active',
        routes: [
            {
                path: '/',
                name: 'summary',
                component: vueLoader('/assets/vue/components/pages/summary.vue')
            },
            {
                path: '/group/:group_id',
                name: 'group',
                component: vueLoader('/assets/vue/components/pages/group.vue'),
                props: true
            }
        ]
    });

    Vue.use(VueRouter);

    return router;
});
