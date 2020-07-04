This project is a fork of [crunchy-proxy](https://github.com/CrunchyData/crunchy-proxy)
to function as an administrative Postgres jump host. Eventually the goal would
be to limit the servers you could target, as well as to log all user-commands
to the Postgres host for later auditing.

Very much a WIP.

TODO:
* Support for the following message types:
  * CancelRequest
