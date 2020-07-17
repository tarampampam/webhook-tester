<template>
    <header class="navbar navbar-expand flex-column flex-md-row flex-sm-row navbar-dark bg-primary">
        <a href="#" class="navbar-brand mr-0 mr-md-2">
            WebHook Tester
        </a>

        <div class="mr-auto">
            <ul class="navbar-nav flex-row d-none d-sm-block">
                <li class="nav-item">
                    <a class="nav-link" href="#"><i class="fas fa-question mr-1"></i> Help</a>
                </li>
            </ul>
        </div>

        <div class="form-inline my-2 my-lg-0">
            <button class="btn btn-success my-2 my-sm-0 border-0" v-bind:data-clipboard-text="currentWebHookUrl">
                <i class="fas fa-copy mr-1"></i> Copy URL
            </button>
            <button class="btn btn-info my-2 ml-2 my-sm-0 border-0">
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
    }
</script>

<style scoped>
</style>
