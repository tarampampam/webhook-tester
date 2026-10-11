export interface APIError extends Error {
  readonly response?: Response
  readonly description: string
}

abstract class BaseAPIError extends Error implements APIError {
  public readonly response?: Response
  public abstract readonly description: string

  constructor({ message, response }: { message?: string; response?: Response } = {}) {
    super(message)

    this.response = response
  }
}

export class APIErrorCommon extends BaseAPIError {
  public readonly description = "Something went wrong on the server side, but we can't identify it as a specific error"
}

export class APIErrorBadRequest extends BaseAPIError {
  public readonly description = 'Bad request'
}

export class APIErrorNotFound extends BaseAPIError {
  public readonly description = 'Not found'
}
