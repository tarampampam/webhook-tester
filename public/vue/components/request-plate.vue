<template>
    <div class="request-plate list-group-item list-group-item-action flex-column py-3 px-3" :class="methodClass">
        <div class="d-flex w-100 justify-content-between">
            <h5 class="mb-1 text-nowrap">{{ clientAddress }}</h5>
            <button type="button" class="close position-relative" title="Delete" @click="remove">&times;</button>
        </div>
        <p class="when small m-0">{{ formattedWhen }}</p>
    </div>
</template>

<script>
    /* global module */

    'use strict';

    module.exports = {
        props: {
            uuid: {
                type: String,
                default: null,
            },
            clientAddress: {
                type: String,
                default: 'X.X.X.X',
            },
            method: {
                type: String,
                default: '',
            },
            when: {
                type: Date,
                default: null,
            },
        },

        data: function () {
            return {
                intervalId: null,
                formattedWhen: '',
            }
        },

        mounted: function () {
            this.updateFormattedWhen();

            this.intervalId = setInterval(() => this.updateFormattedWhen(), 150);
        },

        beforeDestroy: function () {
            clearInterval(this.intervalId);
        },

        computed: {
            methodClass: function () {
                if (typeof this.method === 'string') {
                    switch (this.method.toLowerCase()) {
                        case 'get':
                            return 'border-success';
                        case 'head':
                            return 'border-light';
                        case 'post':
                            return 'border-primary';
                        case 'put':
                            return 'border-info';
                        case 'patch':
                            return 'border-dark';
                        case 'delete':
                            return 'border-danger';
                        case 'connect':
                            return 'border-warning';
                        case 'options':
                            return 'border-secondary';
                        case 'trace':
                            return 'border-light';
                    }
                }

                return '';
            }
        },

        methods: {
            remove(e) {
                // <https://michaelnthiessen.com/pass-function-as-prop/>
                this.$emit('on-delete', this.uuid);

                e.preventDefault();
                e.stopPropagation();

                this.updateFormattedWhen();
            },

            updateFormattedWhen() {
                this.formattedWhen = this.when != null
                    ? `${this.$moment(this.when).format('h:mm:ss a')} (${this.$moment(this.when).fromNow()})`
                    : '';
            }
        }
    }
</script>

<style scoped>
    .request-plate {
        border-width: 0 0 0 0.3em;
        border-top-color: transparent !important;
        border-right-color: transparent !important;
        border-bottom-color: transparent !important;
    }

    .request-plate .when {
        line-height: 1em;
        opacity: 0.8;
    }

    .request-plate .close {
        top: -5px;
    }
</style>
