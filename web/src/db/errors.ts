/** Custom error class for database errors */
export class DatabaseError extends Error {
  public readonly original?: Error

  constructor(message: string, original?: Error | unknown) {
    super(message)

    if (original instanceof Error) {
      this.original = original
    } else if (original) {
      this.original = new Error(String(original))
    } else {
      this.original = undefined
    }

    this.name = 'DatabaseError'
  }
}
