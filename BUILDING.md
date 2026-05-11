# Building octo-server

## Dependencies

This project depends on several sibling repositories in the OCTO ecosystem:

- [octo-lib](https://github.com/Mininglamp-OSS/octo-lib) — core shared library
- [octo-adapters](https://github.com/Mininglamp-OSS/octo-adapters) — AI agent adapters

While these repositories are private during the pre-release phase,
`go build ./...` may fail with "missing go.sum entry" errors.

## Local build (private preview)

1. Clone the sibling repositories alongside this repo:
   ```
   git clone git@github.com:Mininglamp-OSS/octo-lib.git
   git clone git@github.com:Mininglamp-OSS/octo-server.git
   ```

2. Add a `replace` directive to your local `go.mod`:
   ```
   replace github.com/Mininglamp-OSS/octo-lib => ../octo-lib
   ```

3. Run `go mod tidy && go build ./...`

## Public build

Once all OCTO repositories are public, the standard Go toolchain
will resolve imports from `proxy.golang.org` automatically.

## Docker

```bash
make build          # Builds tangsengdaodaoserver image locally
make run-dev        # docker-compose up full stack
```
