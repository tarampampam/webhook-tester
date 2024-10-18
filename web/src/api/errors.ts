interface APIError extends Error {
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

class APIErrorNotFound extends BaseAPIError {
  public readonly description = 'Not found'
}

class APIErrorCommon extends BaseAPIError {
  public readonly description = "Something went wrong on the server side, but we can't identify it as a specific error"
}

class APIErrorUnknown extends BaseAPIError {
  public readonly description =
    "Something went wrong, and we don't know what (usually on the client or JS libraries side)"
}

export { type APIError, APIErrorNotFound, APIErrorCommon, APIErrorUnknown }
