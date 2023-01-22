# mammoth

> This project is a fork of a fork of [crunchy-proxy](https://github.com/CrunchyData/crunchy-proxy) and 
> still licensed under [Apache 2.0](./LICENSE).
>
> Read about Apache 2.0 [here](https://choosealicense.com/licenses/apache-2.0/).

Mammoth is a proxy/jumpbox for PostgreSQL. It logs all commands sent by clients for auditing purposes. 
For security reasons, it does not log responses from the server.

Mammoth supports:
* SSL (including mTLS, skipping validations, and enforcing SSL as required)
* Arbitrary jumpbox specified by providing the database as "host:port/database"
* Allowable remote hosts can be restricted by a regexp
* User blocking
* Logging of all commands sent to the server
* Query cancellation (by way of parsing server responses and rewriting the "backend secrets")

## Configuring mammoth

You can use one of the example config file and modify it to suit your needs.

### [Config file without SSL](./examples/config.without-ssl.yaml)

```yaml
# Listening address
bind: ":5000"
# Log format (default: json)
logformat: "json"
# Redis host
redisserver: "localhost:6379"
# Configuration when accepting client connections
server:
  # To allow non-SSL upgraded connections (default: false)
  allowUnencrypted: true
# Configuration connecting to backend servers
client:
  # Whether to allow non-SSL connections to the backend (default: false)
  allowUnencrypted: true
  # Whether to attempt an SSL connection at all to the backend (default: true)
  # If this is false, the value of `allowUnencrypted` does not matter
  tryssl: false
```

### [Config file with SSL](./examples/config.with-ssl.yaml)

```yaml
# Listening address
bind: "127.0.0.1:1234"
# Log format (default: json)
logformat: "json"
# Redis host
redisserver: "localhost:6379"
# Configuration when accepting client connections
server:
  # Server certificate
  cert: /etc/pg-jump/server.crt
  # Server key
  key: /etc/pg-jump/server.key
  # CA to verify the client's cert against (if not specified, then
  # client's cert will not be checked, if provided)
  ca: /etc/pg-jump/ca.crt
  # To allow non-SSL upgraded connections (default: false)
  allowUnencrypted: true
# Configuration connecting to backend servers
client:
  # Client certificate
  cert: /etc/pg-jump/client.crt
  # Client key
  key: /etc/pg-jump/client.key
  # CA to verify the server's cert against, if any
  ca: /etc/pg-jump/ca.crt
  # Whether to allow non-SSL connections to the backend (default: false)
  allowUnencrypted: true
  # Whether to attempt an SSL connection at all to the backend (default: true)
  # If this is false, the value of `allowUnencrypted` does not matter
  trySSL: true
```

Then you can run `mammoth` by specifying the path to config file with the `-c` flag.

If you don't specify, `mammoth` will look for the file in `PWD`.

```
mammoth -c <path-to-config>
```

Mammoth also supports the following configuration options:

```
--config|-c <path-to-config-file>
--log-level <trace|debug|info|warn|error|fatal|panic>
--log-format <plain|json>
```

### Managing blocked users  

If you wish to block users instead of managing it directly in the database, you can add 
items to a Redis set named `users`.

An API is being developed to help manage it easily.

For more info about using Redis set, check their [docs](https://redis.io/docs/data-types/sets/).

## Using mammoth

Using mammoth is quite simple. If you're running the server on port `5000`, you can
set mammoth server as the database server host and specify the destination database as the
database name. For example:

```
psql -h mammoth.fqdn.tld -U postgres -p 5000 my_db_server.fqdn.tld/db_name
```
