<template>
    <div>
        <div class="custom-control custom-checkbox d-inline mr-3">
            <input type="checkbox" class="custom-control-input" id="hide-details">
            <label class="custom-control-label" for="hide-details">Hide details</label>
        </div>
        <div class="custom-control custom-checkbox d-inline"
             title="Automatically select and go to the latest incoming webhook request">
            <input type="checkbox" class="custom-control-input" id="auto-navigate" checked>
            <label class="custom-control-label" for="auto-navigate">Auto navigate</label>
        </div>
    </div>
</template>

<script>
    /* global module */

    'use strict';

    module.exports = {
        components: {}
    }
</script>

<style scoped>
</style>
