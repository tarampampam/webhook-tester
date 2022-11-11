const uuidValidationRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/

export function isValidUUID(uuid: string | unknown): boolean {
  return typeof uuid === 'string' && uuidValidationRegex.test(uuid)
}
