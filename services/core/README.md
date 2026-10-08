# Core service (planned)

This separate Go module hosts the private gRPC server. Core owns business rules, MongoDB persistence, the email adapter, and eventually USDA normalization, UTC/Tracking Day handling, and atomic food-log confirmation. It is never exposed directly to the browser.

The first private contract is in `contracts/proto/atethat/v1/identity.proto`. Identity, profile, and preferences have initial implementations; food and health domain services have not begun. Database connection details are configurable so Atlas M0 and a secured self-hosted replica set can use the same Core implementation. Run `go test ./...` from this directory.
