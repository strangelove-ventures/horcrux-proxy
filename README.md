# Horcrux Proxy

horcrux-proxy is a proxy between a horcrux cosigner and one-to-many sentry nodes. This allows the Horcrux cosigner to be kept behind a private network connection so that the only outbound connections are to the other cosigners and the proxy. This allows maintaining the configuration for the list of sentries that horcrux should connect to outside of the private horcrux process. As a benefit, the Horcrux Cosigner does not need to be restarted when adding new sentries for connection.

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

Run the following command to generate the config file:

```bash
horcrux-proxy config init --address tcp://0.0.0.0:1234 --node tcp://10.0.0.10:1234 --node tcp://10.0.0.11:1234 --node tcp://10.0.0.12:1234
```

The following config is generated at `~/.horcrux-proxy/config.yaml`:

```yaml
listenAddr: tcp://0.0.0.0:1234
chainNodes:
- privValAddr: tcp://10.0.0.10:1234
- privValAddr: tcp://10.0.0.11:1234
- privValAddr: tcp://10.0.0.12:1234
```

Start horcrux-proxy

```bash
horcrux-proxy start
```