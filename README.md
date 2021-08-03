# cfspeed

Unofficial CLI-based implementation of [Cloudflare's Speed Test](https://speed.cloudflare.com/)

## Quick start

See https://github.com/makotom/cfspeed/releases for prebuilt binaries.

## Build it for yourself

E.g. if you are to build an executable for Linux on x86-64 (AMD64), you would run:

```
GOOS_LIST_OVERRIDE=("linux") GOARCH_LIST_OVERRIDE=("amd64") ./build-and-pack-all.sh
```

Refer to [the official Go documentation](https://golang.org/doc/install/source#environment) for valid combinations of `GOOS` and `GOARCH`.

Note that the shell script depends on Zip, tar and gzip for packaging.

## Notes

- On Debian/Ubuntu, you will need to install `ca-certificates`. Otherwise errors regarding TLS would be raised.

## TODO

- Go tests
- Smoke tests for AArch64 environments

## Dear Cloudflare

I wrote this application since I sincerely love your Speed Test and [the running rabbit](https://speed.cloudflare.com/static/img/speedrabbit-animate.gif). Why don't we make this official? I'm more than happy to donate these codes.
