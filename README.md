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

## Flags

- `-g`/`--grpc-addr` - address to connect to horcrux via GRPC (preferred over listen addresses since grpc allows multiplexing on a single connection)
- `-l`/`--listen-addr` - add listen address(es) to listen for connection from a horcrux cosigner. If using multiple, it should be to the same cosigner for redundancy. This is deprecated. Use `--grpc-addr` instead.
- `-o`/`--operator` - when true (default), horcrux-proxy will assume it is running in the same kubernetes cluster as sentries deployed with the [cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator). It will use the kube API to discover operator deployments of `type: Sentry` and automatically connect to them.
- `-s`/`--sentry` - sentry(ies) to connect to persistently. If using the [cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator), this is likely not necessary.
- `-a`/`-all` - connect to all sentries regardless of node, instead of only sentries on this node


## Quick Start

If using the [cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator), the required configuration is minimal.

Start command for horcrux-proxy to connect to cosmos operator sentries on the same node:

```bash
horcrux-proxy start -g $HORCRUX_GRPC_ADDR
```

Start command for horcrux-proxy to connect to cosmos operator sentries on all nodes:

```bash
horcrux-proxy start -g $HORCRUX_GRPC_ADDR -a
```

Start command for horcrux-proxy to connect to sentries that are not deployed using cosmos-operator:

```bash
horcrux-proxy start -o=false -g $HORCRUX_GRPC_ADDR -s $SENTRY_1 -s $SENTRY_2 ...
```
