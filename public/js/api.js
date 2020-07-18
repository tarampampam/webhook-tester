'use strict';

/**
 * @typedef {{ip: String, hostname: String, method: String, content: String, headers:Object.<string, string>, url: String, created_at_unix: Number}} recordedRequest
 */

define(['axios', 'session'], (axios, session) => {
    /**
     * @returns {String}
     */
    const getApiUri = () => {
        let loc = window.location,
            result = loc.protocol + '//' + loc.hostname;

        if (loc.port !== '80' && loc.port !== '433') {
            result += ':' + loc.port;
        }

        return result + '/api';
    };

    /**
     * @returns {Promise<{version: String, limits: {max_requests: Number}}>}
     */
    const getAppSettings = () => {
        return new Promise((resolve, reject) => {
            axios
                .get(`${getApiUri()}/settings`)
                .then((response) => resolve(response.data))
                .catch((err) => reject(err));
        });
    };

    /**
     * @returns {Promise<{uuid: String, response: {content: String, code: Number, content_type: String, delay_sec: Number, created_at_unix: Number}}>}
     */
    const startNewSession = () => {
        return new Promise((resolve, reject) => {
            axios
                .post(`${getApiUri()}/session`)
                .then((response) => resolve(response.data))
                .catch((err) => reject(err));
        });
    };

    /**
     * @param {String} uuid
     *
     * @returns {Promise<{success: Boolean}>}
     */
    const deleteSession = (uuid) => {
        return new Promise((resolve, reject) => {
            axios
                .delete(`${getApiUri()}/session/${uuid}`)
                .then((response) => resolve(response.data))
                .catch((err) => reject(err));
        });
    };

    /**
     * @param {String} uuid
     *
     * @returns {Promise<Object.<string, recordedRequest>>}
     */
    const getAllSessionRequests = (uuid) => {
        return new Promise((resolve, reject) => {
            axios
                .get(`${getApiUri()}/session/${uuid}/requests`)
                .then((response) => resolve(response.data))
                .catch((err) => reject(err));
        });
    };

    /**
     * @param {String} sessionUUID
     * @param {String} requestUUID
     *
     * @returns {Promise<recordedRequest>}
     */
    const getSessionRequest = (sessionUUID, requestUUID) => {
        return new Promise((resolve, reject) => {
            axios
                .get(`${getApiUri()}/session/${sessionUUID}/requests/${requestUUID}`)
                .then((response) => resolve(response.data))
                .catch((err) => reject(err));
        });
    };

    /**
     * @param {String} sessionUUID
     * @param {String} requestUUID
     *
     * @returns {Promise<{success: Boolean}>}
     */
    const deleteSessionRequest = (sessionUUID, requestUUID) => {
        return new Promise((resolve, reject) => {
            axios
                .delete(`${getApiUri()}/session/${sessionUUID}/requests/${requestUUID}`)
                .then((response) => resolve(response.data))
                .catch((err) => reject(err));
        });
    };

    /**
     * @param {String} uuid
     *
     * @returns {Promise<{success: Boolean}>}
     */
    const deleteAllSessionRequests = (uuid) => {
        return new Promise((resolve, reject) => {
            axios
                .delete(`${getApiUri()}/session/${uuid}/requests`)
                .then((response) => resolve(response.data))
                .catch((err) => reject(err));
        });
    };

    return {
        getAppSettings,
        startNewSession,
        deleteSession,
        getAllSessionRequests,
        getSessionRequest,
        deleteSessionRequest,
        deleteAllSessionRequests,
    };
});
