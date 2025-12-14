# plat-kronk

Codec helpers and mediadevices demo.

## Usage

- Demo (no download): `task mediadevices:demo`
- Demo (download/upgrade codecs to `lib/codecs` and prefer them): `task mediadevices:demo:install`
- Codec status/install: see `taskfiles/Taskfile.codecs.yml` (e.g., `task codecs:status`, `task codecs:install`)

Without Task:
- `GOWORK=off go run ./cmd/mediadevices-demo`
- `GOWORK=off go run ./cmd/mediadevices-demo -install-codecs -upgrade -lib ./lib/codecs`
