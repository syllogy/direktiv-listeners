# SMTP Listener

## Build
To build this server a simple go command will suffice.

```sh
CGO_ENABLED=0 go build -o smtp-receiver -ldflags="-s -w" *.go
```

## Run
To run this server you will need a config file and a binary which can obtain from follow the build commands above.

Running the following command will start the server

```sh
./smtp-receiver config.yaml
```

## Configuration

```yaml
smtp:
  address: localhost
  port: 2525
  insecureSkipVerify: true
direktiv:
  endpoint: https://dev.local/api/namespaces/admin/event
  token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwcmVmZXJyZWRfdXNlcm5hbWUiOiJ0b2tlbi1jMTU4ZGViNy00NWMyLTRlODEtYjg3Yi1kYzRhZjBkY2FhOTciLCJncm91cHMiOlsidG9rZW4tYzE1OGRlYjctNDVjMi00ZTgxLWI4N2ItZGM0YWYwZGNhYTk3Il0sImV4cCI6MTYyNzQ0MjExMCwiaXNzIjoiZGlyZWt0aXYifQ.T-txuqkh_L_R2tv3Po3MvMf7Rh6kbJf4xqV80Udw6Yk
internal: vorteil.io
```

- internal: the sub domain of the email addresses you would like to send the password to
- direktiv:
  - endpoint: The full endpoint to send a event request on a namespace
  - token: The access token to provide proper authentication with Direktiv
- smtp:
  - address: The address the smtp server will host on
  - port: The port the smtp server will use
  - insecureSkipVerify: skip ssl address checking


