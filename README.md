# mammoth

> This project is a fork of [crunchy-proxy](https://github.com/CrunchyData/crunchy-proxy) and still licensed under [Apache 2.0](./LICENSE).
>
> For more info about Apache 2.0, read more about it [here](https://choosealicense.com/licenses/apache-2.0/).

Mammoth logs all commands sent by clients for auditing purposes. It does not log
responses from the server, for data-protection reasons.

Supports:
* SSL (including mTLS, skipping validations, and enforcing SSL as required)
* Arbitrary jump-host specified by providing the database as "host:port/database"
* Allowable remote hosts can be restricted by a regexp
* Logging of all commands (except auth, of course) sent to the server
* Query cancellation (by way of parsing server responses and rewriting the "backend secrets")

## Building

Install a recent version of Go (I built this on 1.13), then:

```
go build .
```

The output executable will be in the root directory with the name `pg-jump`.

## Usage

Copy one of the example configs ([without SSL](./examples/config.without-ssl.yaml),
or [with SSL](./examples/config.with-ssl.yaml)) and modify it to suit. Then you
can run the `pg-jump` executable. By default it looks in the PWD for
`config.yaml` and loads those values. Otherwise, you can specify:

```
pg-jump -c <path-to-config>
```

You probably want to capture the stdout output somewhere for later auditing.

The program supports the following configuration options:

```
--config|-c <path-to-config-file>
--log-level <trace|debug|info|warn|error|fatal|panic>
--log-format <plain|json>
```

Clients can now connect to the pg-jump host. Specify the database as a complete
host, e.g., if pg-jump is running on port `5000` (`bind: :5000` in the
`config.yaml` file) and you have another database running on port `5432`:

```
psql -h localhost -U postgres -p 5000 localhost:5432/db-name
```
