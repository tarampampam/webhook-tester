'use strict';

define([], () => {
    /**
     * @type {String}
     */
    const storageSessionUuidKey = 'session_uuid';

    /**
     * @returns {String|null}
     */
    const getLocalSessionUUID = function () {
        return localStorage.getItem(storageSessionUuidKey);
    };

    /**
     * @param {String} uuid
     */
    const setLocalSessionUUID = function (uuid) {
        localStorage.setItem(storageSessionUuidKey, uuid);
    }

    return {
        getLocalSessionUUID,
        setLocalSessionUUID,
    };
});
