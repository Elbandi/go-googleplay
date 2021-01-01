# [WIP] Google Playstore Client

Binaries are [here](https://github.com/Jarijaas/go-googleplay/releases)

Terminology:

* Google Services Framework ID (GSFID) - Device specific identifier (hex string)
    * Generated against a device config send to Google servers
* AuthSub Token - Authentication token to Playstore, exchanged against account email and password

```
Client for Google Playstore, can download apps

Usage:
  gplay [flags]
  gplay [command]

Available Commands:
  download    Download app
  help        Help about any command
  login       Login using the credentials, returns new or cached gsfId and authSub

Flags:
      --authSub string    Alternatively, set env var GPLAY_AUTHSUB
      --email string
      --force-login       Authenticate, even if current gsfId and authSubToken are valid
      --gsfId string      Alternatively, set env var GPLAY_GSFID
  -h, --help              help for gplay
      --password string
  -v, --verbose           Enable debug messages

Use "gplay [command] --help" for more information about a command.
```
