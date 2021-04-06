'use strict';

/* global define */

define(['reconnectingWebsocket'], (rws) => {
    /**
     * @returns {String}
     */
    const getWebsocketBaseUri = () => {
        const loc = window.location;

        let result = (loc.protocol === 'http:' ? 'ws' : 'wss') + '://' + loc.hostname;

        if (loc.port !== '80' && loc.port !== '433') {
            result += ':' + loc.port;
        }

        return result + '/ws';
    };

    /**
     * @param {String} sessionUUID
     * @param {function(name: string, data: string): void} onMessage
     *
     * @returns {WebSocket}
     */
    const newSessionConnection = (sessionUUID, onMessage) => {
        const ws = new WebSocket(getWebsocketBaseUri() + '/session/' + sessionUUID);

        ws.onmessage = (message) => {
            const j = JSON.parse(message.data);

            onMessage(j.name, j.data);
        };

        return ws;
    };

    /**
     * @param {String} sessionUUID
     * @param {function(name: string, data: string): void} onMessage
     *
     * @returns {ReconnectingWebSocket}
     */
    const newRenewableSessionConnection = (sessionUUID, onMessage) => {
        const ws = new rws(getWebsocketBaseUri() + '/session/' + sessionUUID, null, {
            automaticOpen: true,
            reconnectInterval: 1000,
            maxReconnectInterval: 10000,
        });

        ws.onmessage = (message) => {
            const j = JSON.parse(message.data);

            onMessage(j.name, j.data);
        };

        return ws;
    };

    return {
        getWebsocketBaseUri,
        newSessionConnection,
        newRenewableSessionConnection,
    };
});
