<template>
    <div class="request-plate list-group-item list-group-item-action flex-column">
        <div class="d-flex w-100 justify-content-between">
            <h5 class="mb-1 text-nowrap">{{ ip }}<span
                class="badge text-uppercase ml-2 http-method"
                :class="methodClass"
                :v-if="method"
            >{{ method }}</span>
            </h5>
            <button type="button" class="close" title="Delete" @click="remove">&times;</button>
        </div>
        <small>{{ formattedWhen }}</small>
    </div>
</template>

<script>
    /* global module */

    'use strict';

    module.exports = {
        props: {
            uuid: {
                type: String,
                default: undefined,
            },
            ip: {
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

            this.intervalId = setInterval(() => this.updateFormattedWhen(), 500);
        },

        beforeDestroy: function () {
            clearInterval(this.intervalId);
        },

        computed: {
            methodClass: function () {
                if (typeof this.method === 'string') {
                    switch (this.method.toLowerCase()) {
                        case 'get':
                            return 'badge-success';
                        case 'post':
                        case 'put':
                            return 'badge-info';
                        case 'delete':
                            return 'badge-danger';
                    }
                }

                return 'badge-light';
            }
        },
        methods: {
            remove() {
                // <https://michaelnthiessen.com/pass-function-as-prop/>
                this.$emit('on-delete', this.uuid);

                //this.$destroy();
                //this.$el.parentNode.removeChild(this.$el);
            },

            updateFormattedWhen() {
                this.formattedWhen = this.when != null
                    ? `${this.$moment(this.when).format('YYYY-MM-D h:mm:ss a')} (${this.$moment(this.when).fromNow()})`
                    : '';
            }
        }
    }
</script>

<style scoped>
    .request-plate {
        cursor: pointer;
    }

    .http-method {
        position: relative;
        top: -.15em;
    }
</style>
