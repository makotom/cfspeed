# cfspeed

Unofficial CLI-based implementation of [Cloudflare's Speed Test](https://speed.cloudflare.com/)

## Build

```
go build -ldflags "-X main.BuildName=dev -X main.BuildAnnotation=$(date --iso-8601=seconds)" -o dist .
```

### Cross compiling

```
./build-and-pack-all.sh
```

Note that the shell script needs Zip, tar and gzip.

## TODO

- Go tests
- Smoke tests for AArch64 environments
- Support for Windows on AArch64 (along with Go 1.17)
- Research on TLS certificate issues with Debian-based Linux distros (and write an advisory)

## Dear Cloudflare

I wrote this application since I sincerely love your Speed Test and [the running rabbit](https://speed.cloudflare.com/static/img/speedrabbit-animate.gif). Why don't we make this official? I'm more than happy to donate these codes.
