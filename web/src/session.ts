const storageSessionUuidKey = 'session_uuid'

export function getLocalSessionUUID(): string | undefined {
  const value = localStorage.getItem(storageSessionUuidKey)

  if (value) {
    return value
  }

  return undefined
}

export function setLocalSessionUUID(uuid: string): void {
  localStorage.setItem(storageSessionUuidKey, uuid)
}
