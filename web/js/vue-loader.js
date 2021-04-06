'use strict';

/* global define */

define(['axios', 'httpVueLoader', 'Vue'], (axios, httpVueLoader, Vue) => {
    // @link <https://github.com/FranckFreiburger/http-vue-loader/#api>
    httpVueLoader.httpRequest = (url) => {
        return axios.get(url, { timeout: 10000 })
            .then((res) => {
                return res.data;
            })
            .catch((err) => {
                console.error(err);
                return Promise.reject(err.status);
            });
    };

    Vue.use(httpVueLoader);

    return httpVueLoader;
});
