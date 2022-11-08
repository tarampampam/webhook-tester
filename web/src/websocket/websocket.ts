import ReconnectingWebSocket from 'reconnecting-websocket'

function getWebsocketBaseUri(): string{
  const loc = window.location

  let result = (loc.protocol === 'http:' ? 'ws' : 'wss') + '://' + loc.hostname

  if (loc.port !== '80' && loc.port !== '433') {
    result += ':' + loc.port
  }

  return result + '/ws'
}

type WebsocketHandler = (name: string, data: string) => void

export function newRenewableSessionConnection(sessionUUID: string, onMessage: WebsocketHandler): ReconnectingWebSocket {
  const ws = new ReconnectingWebSocket(getWebsocketBaseUri() + '/session/' + sessionUUID, undefined, {
    maxReconnectionDelay: 10000,
  })

  ws.addEventListener('message', (msg) => {
    const j = JSON.parse(msg.data)

    onMessage(j.name, j.data)
  })

  return ws
}
