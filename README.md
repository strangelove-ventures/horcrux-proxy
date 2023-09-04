# Horcrux Proxy

horcrux-proxy is a proxy between a horcrux cosigner and one-to-many sentry nodes. This allows the Horcrux cosigner to be kept behind a private network connection so that the only outbound connections are to the other cosigners and the proxy. 

This allows maintaining the configuration for the list of sentries that horcrux should connect to outside of the private horcrux process. As a benefit, the Horcrux Cosigner does not need to be restarted when new sentries are created.

Additionally, horcrux-proxy will watch the kubernetes cluster for [cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator) sentries so that the proxy does not need to be restarted when sentries are added or removed.

## Diagram

```
               +

               +                    +------------+
                                    |            |
               +             +----->|  Chain A   |
                             |      |   Sentry   |
               +             |      +------------+
                             |
+------------+ +   +---------+---+  +------------+
|            |     |             |  |            |
|  Horcrux   +---->|   Horcrux   +->|  Chain B   |
|  Cosigner  |     |    Proxy    |  |   Sentry   |
+------------+ +   +---------+---+  +------------+
                             |
               +             |      +------------+
                             |      |            |
               +             +----->|  Chain N   |
                                    |   Sentry   |
               +                    +------------+

               +
```

## Quick Start

Start horcrux-proxy

```bash
horcrux-proxy start
```

### Flags

- `-l`/`--listen-addr` - modify listen address (default `tcp://0.0.0.0:1234`)
- `-a`/`-all` - connect to all sentries regardless of node, instead of only sentries on this node