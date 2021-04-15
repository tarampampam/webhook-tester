'use strict';

/* global define, hljs */

define(['Vue', 'hljs'], (Vue) => {
    // <https://vuejsfeed.com/blog/vue-js-syntax-highlighting-with-highlight-js>
    Vue.directive('highlightjs', {
        deep: true,
        bind: function (el, binding) {
            el.querySelectorAll('code').forEach((target) => {
                if (binding.value) {
                    target.className = 'hljs' // reset highlighting language
                    target.innerText = binding.value;
                }
                hljs.highlightElement(target);
            })
        },
        componentUpdated: function (el, binding) {
            el.querySelectorAll('code').forEach((target) => {
                if (binding.value) {
                    target.className = 'hljs' // reset highlighting language
                    target.innerText = binding.value;
                    hljs.highlightElement(target);
                }
            })
        }
    });
});
