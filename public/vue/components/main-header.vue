<template>
    <header class="navbar navbar-expand flex-column flex-md-row flex-sm-row navbar-dark bg-primary">
        <span class="navbar-brand mr-0 mr-md-2">
            WebHook Tester
        </span>

        <div class="mr-auto">
            <ul class="navbar-nav flex-row d-none d-sm-block">
                <li class="nav-item">
                    <span class="nav-link"><i class="fas fa-question mr-1"></i> Help</span>
                </li>
            </ul>
        </div>

        <div class="form-inline my-2 my-lg-0">
            <button class="btn btn-success my-2 my-sm-0 border-0" v-bind:data-clipboard-text="currentWebHookUrl">
                <i class="fas fa-copy mr-1"></i> Copy URL
            </button>
            <button class="btn btn-info my-2 ml-2 my-sm-0 border-0" @click="newURL">
                <i class="fas fa-plus mr-1"></i> New URL
            </button>
        </div>
    </header>
</template>

<script>
    /* global module */

    'use strict';

    module.exports = {
        props: {
            currentWebHookUrl: {
                type: String,
                default: 'URL was not defined',
            },
        },

        methods: {
            newURL() {
                // <https://michaelnthiessen.com/pass-function-as-prop/>
                this.$emit('on-new-url');
            },
        }
    }
</script>

<style scoped>
    .nav-link {
        cursor: pointer;
    }
</style>
