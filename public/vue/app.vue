<template>
    <div class="">
        Hello world!
    </div>
</template>

<script>
    /* global module */

    'use strict';

    module.exports = {
        components: {},
        mounted() {
            document.getElementById('main-loader').remove();
        }
    }
</script>

<style scoped>
</style>
