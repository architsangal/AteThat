# AteThat contracts

`openapi/v1.yaml` is the public HTTP contract. `proto/atethat/v1/identity.proto` is the private API-to-Core contract for the first implementation slice. Both follow `docs/5. API_DESIGN.md`; that design document remains the complete Phase 1 endpoint inventory until every vertical slice has been formalized here.

These files currently cover **local-email identity**, Google sign-in and explicit linking, profile, and preferences. Account deletion remains in the design document until its grace period is fixed. Food, nutrition, meal, health, and history routes have not yet been specified as executable contracts. Extend both contracts together as each vertical slice is implemented. Do not use untyped JSON envelopes to bypass the contract.

The browser calls only the REST API. The API service calls Core over private gRPC. Contract files can be consumed by both applications, but generated code belongs inside each consumer's own module; there is no shared runtime helper package. The proto deliberately omits a Go package path so each service can supply its own code-generation mapping.

Run `make contracts-check` from the repository root to parse OpenAPI, validate local references and operation IDs, and compile the Protocol Buffer descriptor. Code generation will be added when the Go services are implemented.
