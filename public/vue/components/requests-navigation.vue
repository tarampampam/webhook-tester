<template>
    <div>
        <div class="btn-group" role="group">
            <button type="button" class="btn btn-secondary btn-sm">First request</button>
            <button type="button" class="btn btn-secondary btn-sm">
                <i class="fas fa-arrow-left"></i> Previous
            </button>
        </div>
        <div class="btn-group" role="group">
            <button type="button" class="btn btn-secondary btn-sm">
                Next <i class="fas fa-arrow-right"></i>
            </button>
            <button type="button" class="btn btn-secondary btn-sm">Last request</button>
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
