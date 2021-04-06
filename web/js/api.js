'use strict';

/* global define */

define(['axios'], (axios) => {
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
     * @returns {Promise<APIVersion>}
     */
    const getAppVersion = () => {
        return new Promise((resolve, reject) => {
            axios
                .get(`${getApiUri()}/version`)
                .then((response) => resolve(response.data))
                .catch((err) => reject(err));
        });
    };

    /**
     * @returns {Promise<APISettings>}
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
     * @param {APINewSessionSettings} settings
     *
     * @returns {Promise<APINewSession>}
     */
    const startNewSession = (settings) => {
        return new Promise((resolve, reject) => {
            axios
                .post(`${getApiUri()}/session`, settings)
                .then((response) => resolve(response.data))
                .catch((err) => reject(err));
        });
    };

    /**
     * @param {String} uuid
     *
     * @returns {Promise<APIDeleteSession>}
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
     * @returns {Promise<APIRecordedRequest[]>}
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
     * @returns {Promise<APIRecordedRequest>}
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
     * @returns {Promise<APIDeleteSessionRequest>}
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
     * @returns {Promise<APIDeleteAllSessionRequests>}
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
        getAppVersion,
        getAppSettings,
        startNewSession,
        deleteSession,
        getAllSessionRequests,
        getSessionRequest,
        deleteSessionRequest,
        deleteAllSessionRequests,
    };
});
