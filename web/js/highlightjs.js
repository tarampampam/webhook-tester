'use strict';

/* global define, hljs */

define(['Vue', 'hljs'], (Vue) => {
    /**
     * @param {String} code
     */
    const formatSourceCode = (code) => {
        try { // decorate json
            return JSON.stringify(JSON.parse(code), null, 2);
        } catch (e) {
            // wrong json
        }

        return code; // fallback
    };

    // <https://vuejsfeed.com/blog/vue-js-syntax-highlighting-with-highlight-js>
    Vue.directive('highlightjs', {
        deep: true,
        bind: function (el, binding) {
            el.querySelectorAll('code').forEach((target) => {
                if (binding.value) {
                    target.className = 'hljs' // reset highlighting language
                    target.innerText = formatSourceCode(binding.value);
                }
                hljs.highlightElement(target);
            })
        },
        componentUpdated: function (el, binding) {
            el.querySelectorAll('code').forEach((target) => {
                if (binding.value) {
                    target.className = 'hljs' // reset highlighting language
                    target.innerText = formatSourceCode(binding.value);
                    hljs.highlightElement(target);
                }
            })
        }
    });
});
