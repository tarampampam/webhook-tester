import ReconnectingWebSocket from 'reconnecting-websocket'

function getWebsocketBaseUri(): string{
  const loc = window.location

  let result = (loc.protocol === 'http:' ? 'ws' : 'wss') + '://' + loc.hostname

  if (loc.port !== '80' && loc.port !== '433') {
    result += ':' + loc.port
  }

  return result + '/ws'
}

export function newRenewableSessionConnection(sessionUUID: string, handlers: {
  onRequestRegistered?: (requestUUID: string) => void,
  onRequestDeleted?: (requestUUID: string) => void,
  onRequestsDeleted?: () => void,
}): ReconnectingWebSocket {
  const ws = new ReconnectingWebSocket(getWebsocketBaseUri() + '/session/' + sessionUUID, undefined, {
    maxReconnectionDelay: 10000,
  })

  ws.addEventListener('message', (msg): void => {
    const j = JSON.parse(msg.data) as {name: 'request-registered' | 'request-deleted' | 'requests-deleted', data: unknown}

    switch (j.name) {
      case 'request-registered':
        if (handlers.onRequestRegistered && typeof j.data === 'string') {
          handlers.onRequestRegistered(j.data)
        }
        break

      case 'request-deleted':
        if (handlers.onRequestDeleted && typeof j.data === 'string') {
          handlers.onRequestDeleted(j.data)
        }
        break

      case 'requests-deleted':
        if (handlers.onRequestsDeleted) {
          handlers.onRequestsDeleted()
        }
        break
    }
  })

  return ws
}
