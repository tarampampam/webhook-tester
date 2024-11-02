/** Convert a session ID to a URL. */
export function sessionToUrl(sID: string): URL {
  return new URL(`${window.location.origin}/${sID}`)
}
