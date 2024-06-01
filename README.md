# TorMan

TorMan is a simple application that manages multiple Tor clients. 
```
Usage: torman start [base port] [public port]
       torman stop
```

The `start` command expects a `base port` as argument. New clients will be will be spawned in sequence, starting from the `base port`, increasing by `1` for each client. `TorMan` will start one client for each CPU core. The clients will only be available locally but `TorMan` exposes a loadbalancer at `public port` that can be used by other machines.

Behind the scenes `TorMan` creates `torrc` config files and `systemd` services which it controls. On every start it will stop all running instances, purge their data and start afresh.
