# pg-jump

This project is a fork of [crunchy-proxy](https://github.com/CrunchyData/crunchy-proxy)
to function as an administrative Postgres jump host with audit logging.

It logs all commands sent by clients for auditing purposes. It does not log
responses from the server, for data-protection reasons.

Authentication is merely a pass-thru to the backend Postgres instance, and is
not handled by this proxy server. This is currently a non-goal, but may
eventually become a goal.

Supports:
* SSL (including mTLS, skipping validations, and enforcing SSL as required)
* Arbitrary jump-host specified by providing the database as "host:port/database"
* Logging of all commands (except auth, of course) sent to the server
* Query cancellation (by way of parsing server responses and rewriting the "backend secrets")

TODO:
* Restrict the set of allowable target hosts via a list or regex

## Ps

This is not the finest code. It is the functionaly-ist code. There aren't
tests. This product may explode at any time. No warranties, no returns.
Please have your pets spayed and neutered.
