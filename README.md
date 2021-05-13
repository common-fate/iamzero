# iamzero

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for information on how to contribute and set up a local version for development.

## Authentication

Currently simple token-based authentication has been implemented. When booting the server set the IAMZERO_TOKEN environment variable to a randomly generated string. Requests to the server will require the `x-iamzero-token` header to be set to this value, otherwise a HTTP 401 unauthorized response will be returned. The admin console will also prompt for this token when first launched.

Improved authentication and authorization, including support and auditing for multiple administrators and instrumented services, is planned in the iamzero roadmap.
