# pg-jump: a postgres jumphost with query audit-logging

This project is a fork of [crunchy-proxy](https://github.com/CrunchyData/crunchy-proxy)
to function as an administrative Postgres jump host with audit logging. It is
useful when your administrators need direct access to databases on occasion,
but compliance requires that you have visibility into their actions.

It logs all commands sent by clients for auditing purposes. It does not log
responses from the server, for data-protection reasons.

Authentication is merely a pass-thru to the backend Postgres instance, and is
not handled by this proxy server. This is currently a non-goal, but may
eventually become a goal.

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

## Ps

This is not the finest code. It is the functionaly-ist code. There aren't
tests. This product may explode at any time. No warranties, no returns.
Please have your pets spayed and neutered.
