'use strict';

/* global define */

/**
 * @typedef {Object} AxiosResponse
 * @property {Object} config
 * @property {Object|Array|any} data
 * @property {Object.<string, string>} headers
 * @property {XMLHttpRequest} request
 * @property {number} status
 * @property {string} statusText
 */

define(['axios', 'Base64'], (axios, Base64) => {
    /** @type {TextDecoder} */
    const textDecoder = new TextDecoder("utf-8");

    /** @type {TextEncoder} */
    const textEncoder = new TextEncoder();

    /**
     * @returns {String}
     */
    const getApiUri = () => {
        const loc = window.location;
        let result = loc.protocol + '//' + loc.hostname;

        if (loc.port !== '80' && loc.port !== '433') {
            result += ':' + loc.port;
        }

        return result + '/api';
    };

    /**
     * @typedef {Object} APIVersion
     * @property {String} version
     */
    /**
     * @returns {Promise<APIVersion>}
     */
    const getAppVersion = () => {
        return new Promise((resolve, reject) => {
            axios
                .get(`${getApiUri()}/version`)
                .then(/** @param {AxiosResponse} response */(response) => {
                    /** @type {{version: string}} */
                    const data = response.data;

                    /** @type {APIVersion} */
                    const result = {version: data.version};

                    resolve(result);
                })
                .catch((err) => reject(err));
        });
    };

    /**
     * @typedef {Object} APISettings
     * @property {{sessionLifetimeSec: number, maxRequests: number, maxWebhookBodySize: number}} limits
     */
    /**
     * @returns {Promise<APISettings>}
     */
    const getAppSettings = () => {
        return new Promise((resolve, reject) => {
            axios
                .get(`${getApiUri()}/settings`)
                .then(/** @param {AxiosResponse} response */(response) => {
                    /** @type {{
                     *    limits: {
                     *      session_lifetime_sec: number,
                     *      max_requests: number,
                     *      max_webhook_body_size: number,
                     *    }
                     *  }} */
                    const data = response.data;

                    /** @type {APISettings} */
                    const result = {
                        limits: {
                            sessionLifetimeSec: data.limits.session_lifetime_sec,
                            maxRequests: data.limits.max_requests,
                            maxWebhookBodySize: data.limits.max_webhook_body_size,
                        },
                    };

                    resolve(result);
                })
                .catch((err) => reject(err));
        });
    };

    /**
     * @typedef {Object} APINewSessionSettings
     * @property {Number} [statusCode]
     * @property {String} [contentType]
     * @property {Number} [responseDelay]
     * @property {Uint8Array} [responseContent]
     */
    /**
     * @typedef {Object} APINewSession
     * @property {String} UUID
     * @property {{content: Uint8Array, code: Number, contentType: string, delaySec: Number}} response
     * @property {Date} createdAt
     */
    /**
     * @param {APINewSessionSettings} settings
     *
     * @returns {Promise<APINewSession>}
     */
    const startNewSession = (settings) => {
        return new Promise((resolve, reject) => {
            /** @type {{
             *    status_code: ?number,
             *    content_type: ?string,
             *    response_delay: ?number,
             *    response_content_base64: ?string,
             *  }} */
            const postSettings = {};

            if (typeof settings.statusCode === "number") {
                postSettings.status_code = settings.statusCode;
            }

            if (typeof settings.responseDelay === "number") {
                postSettings.response_delay = settings.responseDelay;
            }

            if (typeof settings.contentType === "string") {
                postSettings.content_type = settings.contentType;
            }

            if (typeof settings.responseContent === "object") {
                // convert {ArrayBuffer} into base64-encoded string
                postSettings.response_content_base64 = Base64.encode(textDecoder.decode(settings.responseContent));
            }

            console.log(settings, postSettings);

            axios
                .post(`${getApiUri()}/session`, postSettings)
                .then(/** @param {AxiosResponse} response */(response) => {
                    /** @type {{
                     *    uuid: string,
                     *    response: {
                     *      content_base64: string,
                     *      content_type: string,
                     *      code: number,
                     *      delay_sec: number,
                     *    },
                     *    created_at_unix: number,
                     *  }} */
                    const data = response.data;

                    /** @type {APINewSession} */
                    const result = {
                        UUID: data.uuid,
                        response: {
                            contentType: data.response.content_type,
                            code: data.response.code,
                            delaySec: data.response.delay_sec,
                            content: textEncoder.encode(Base64.decode(data.response.content_base64)),
                        },
                        createdAt: new Date(data.created_at_unix * 1000),
                    };

                    resolve(result);
                })
                .catch((err) => reject(err));
        });
    };

    /**
     * @typedef {Object} APIDeleteSession
     * @property {Boolean} success
     */
    /**
     * @param {String} uuid
     *
     * @returns {Promise<APIDeleteSession>}
     */
    const deleteSession = (uuid) => {
        return new Promise((resolve, reject) => {
            axios
                .delete(`${getApiUri()}/session/${uuid}`)
                .then(/** @param {AxiosResponse} response */(response) => {
                    /** @type {{ success: boolean }} */
                    const data = response.data;

                    /** @type {APIDeleteSession} */
                    const result = {
                        success: data.success,
                    };

                    resolve(result);
                })
                .catch((err) => reject(err));
        });
    };

    /**
     * @typedef {Object} APIRecordedRequest
     * @property {String} UUID
     * @property {String} clientAddress
     * @property {String} method
     * @property {Uint8Array} content
     * @property {{name: string, value: string}[]} headers
     * @property {String} url - relative (`/foo/bar`, NOT `http://example.com/foo/bar`)
     * @property {Date} createdAt
     */
    /**
     * @param {String} sessionUUID
     * @param {String} requestUUID
     *
     * @returns {Promise<APIRecordedRequest>}
     */
    const getSessionRequest = (sessionUUID, requestUUID) => {
        return new Promise((resolve, reject) => {
            axios
                .get(`${getApiUri()}/session/${sessionUUID}/requests/${requestUUID}`)
                .then(/** @param {AxiosResponse} response */(response) => {
                    /** @type {{
                     *    uuid: string,
                     *    client_address: string,
                     *    method: string,
                     *    content_base64: string,
                     *    headers: {name: string, value: string}[],
                     *    url: string,
                     *    created_at_unix: number,
                     *  }} */
                    const data = response.data;

                    /** @type {APIRecordedRequest} */
                    const result = {
                        UUID: data.uuid,
                        clientAddress: data.client_address,
                        method: data.method,
                        content: textEncoder.encode(Base64.decode(data.content_base64)),
                        headers: data.headers,
                        url: data.url,
                        createdAt: new Date(data.created_at_unix * 1000),
                    };

                    resolve(result);
                })
                .catch((err) => reject(err));
        });
    };

    /**
     * @param {String} uuid
     *
     * @returns {Promise<APIRecordedRequest[]>}
     */
    const getAllSessionRequests = (uuid) => {
        return new Promise((resolve, reject) => {
            axios
                .get(`${getApiUri()}/session/${uuid}/requests`)
                .then(/** @param {AxiosResponse} response */(response) => {
                    /** @type {{
                     *    uuid: string,
                     *    client_address: string,
                     *    method: string,
                     *    content_base64: string,
                     *    headers: {name: string, value: string}[],
                     *    url: string,
                     *    created_at_unix: number,
                     *  }[]} */
                    const data = response.data;

                    /** @type {APIRecordedRequest[]} */
                    const result = [];

                    data.forEach((item) => {
                        result.push({
                            UUID: item.uuid,
                            clientAddress: item.client_address,
                            method: item.method,
                            content: textEncoder.encode(Base64.decode(item.content_base64)),
                            headers: item.headers,
                            url: item.url,
                            createdAt: new Date(item.created_at_unix * 1000),
                        })
                    })

                    resolve(result);
                })
                .catch((err) => reject(err));
        });
    };

    /**
     * @typedef {Object} APIDeleteSessionRequest
     * @property {Boolean} success
     */
    /**
     * @param {String} sessionUUID
     * @param {String} requestUUID
     *
     * @returns {Promise<APIDeleteSessionRequest>}
     */
    const deleteSessionRequest = (sessionUUID, requestUUID) => {
        return new Promise((resolve, reject) => {
            axios
                .delete(`${getApiUri()}/session/${sessionUUID}/requests/${requestUUID}`)
                .then(/** @param {AxiosResponse} response */(response) => {
                    /** @type {{ success: boolean }} */
                    const data = response.data;

                    /** @type {APIDeleteSessionRequest} */
                    const result = {
                        success: data.success,
                    };

                    resolve(result);
                })
                .catch((err) => reject(err));
        });
    };

    /**
     * @typedef {Object} APIDeleteAllSessionRequests
     * @property {Boolean} success
     */
    /**
     * @param {String} uuid
     *
     * @returns {Promise<APIDeleteAllSessionRequests>}
     */
    const deleteAllSessionRequests = (uuid) => {
        return new Promise((resolve, reject) => {
            axios
                .delete(`${getApiUri()}/session/${uuid}/requests`)
                .then(/** @param {AxiosResponse} response */(response) => {
                    /** @type {{ success: boolean }} */
                    const data = response.data;

                    /** @type {APIDeleteAllSessionRequests} */
                    const result = {
                        success: data.success,
                    };

                    resolve(result);
                })
                .catch((err) => reject(err));
        });
    };

    return {
        getAppVersion,
        getAppSettings,
        startNewSession,
        deleteSession,
        getSessionRequest,
        getAllSessionRequests,
        deleteSessionRequest,
        deleteAllSessionRequests,
    };
});
