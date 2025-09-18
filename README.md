# Multichat

A lightweight terminal chat room that uses UDP multicast so multiple machines on the same network segment can discover each other without any central server. Every participant joins the same multicast group, sends encoded chat messages, and prints anything they receive in real time.

## Requirements

- Go 1.20 or newer.
- All machines need multicast enabled on their network (most wired/Wi-Fi LANs allow this by default).

## Running

From this directory you can run the tool directly with `go run`:

```sh
go run .
```

Or build a binary once and reuse it:

```sh
go build -o multichat
./multichat
```

When you start the program it will join the default group `239.42.0.1:9999`, announce your arrival, and wait for you to type messages. Anything you type is sent to the multicast group and echoed by anyone else that is running the program on the same group.

### Command-line options

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `-group` | `239.42.0.1` | Multicast group IP to join (IPv4 or IPv6). Defaults to `255.255.255.255` when `-broadcast` is used. |
| `-port` | `9999` | UDP port number shared by every participant. |
| `-name` | system username | Display name shown next to your messages. |
| `-iface` | system default | Specific network interface to bind to (useful on machines with multiple NICs). |
| `-ttl` | `1` | Multicast TTL / hop limit. Increase if a router sits between chat participants. |
| `-broadcast` | `false` | Switch to IPv4 broadcast mode (one-to-many on the local subnet). |

Example with a custom interface and display name:

```sh
go run . -name "Alice" -iface en0

# Broadcast on Wi-Fi using the subnet broadcast address
go run . -broadcast -group 192.168.1.255 -iface en0 -ttl 2
```

Start the program on every machine using the same group and port values. Messages should begin appearing immediately.

## Tips

- `Ctrl+C` or `Ctrl+D` cleanly exits the program.
- Messages longer than the UDP payload limit (about 64 KB with encoding) are rejected locally so they cannot be truncated on the network.
- If you need to span subnets, adjust your router to forward multicast or try a group in the administratively-scoped range (`239.0.0.0/8`).
- When multicast is filtered on your LAN, switch to `-broadcast` with the subnet broadcast address for local-segment delivery.
- On IPv6, supplying `-iface` is often required for link-local multicast groups (e.g. addresses starting with `ff02`).
