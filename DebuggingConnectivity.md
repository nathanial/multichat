# Multicast Send Troubleshooting (Red Hat vs Raspberry Pi)

## Observed behaviour
- Raspberry Pi can send and receive; its messages arrive on the Red Hat box.
- Red Hat instance can receive everything, but messages typed there only appear locally (looped back) and never reach the Pi.
- The program reports no errors because the socket write succeeds from the kernel's point of view.

## What remains suspicious
- Interface selection is already forced via the CLI flag, so the lingering suspects are routing policy, hop limits (TTL), and filtering.
- Many Red Hat kernels still default multicast routes to `lo`; `IP_MULTICAST_IF` overrides that, but the current binary relies on binding to the interface IP, which can leave routing decisions unchanged.
- TTL / hop-limit mismatches can silently stop traffic after the first hop (routers drop TTL 1), yet the sender still sees its own loopback echo.

## Next debugging passes
- `ip route get 239.42.0.1` → confirm the kernel plans to use the physical NIC; if you still see `dev lo`, add a multicast route: `sudo ip route add 239.42.0.0/16 dev enp3s0` (replace interface name).
- `ss -u -g` → verify the socket is bound to the expected local address and joined the multicast group on that interface.
- `sudo tcpdump -ni enp3s0 udp port 9999` (and again on `lo`) → prove whether packets depart the NIC or are stuck on loopback.
- `sudo tcpdump -ni wlan0 udp port 9999` on the Raspberry Pi (replace interface name) → check whether frames ever arrive; if they do, focus on the Pi OS/firewall, otherwise the network gear is dropping them.
- Inspect the TTL in the packet capture (`tcpdump -v`) to confirm it matches what you set with `-ttl` and is not being decremented to zero in transit.
- `ip maddr show dev enp3s0` → ensure the interface subscribed to `239.42.0.1`.
- `sysctl net.ipv4.conf.enp3s0.mc_forwarding` → should be `0`; even so, multicast routers may discard TTL 1 if forwarding is disabled upstream.
- Use `socat` to isolate the transport layer: `socat -v UDP4-RECVFROM:9999,ip-add-membership=239.42.0.1:0,fork -` on the Pi.
- On Red Hat, use interface-based forcing: `socat - UDP4-DATAGRAM:239.42.0.1:9999,ip-multicast-if=enp3s0,ip-multicast-ttl=8` (replace `enp3s0` with your NIC name). Older builds accept device names rather than IP literals.
- If the distribution lacks `ip-multicast-if`, fall back to binding a source: `socat - UDP4-DATAGRAM:239.42.0.1:9999,bind=192.168.1.20:0,ip-multicast-ttl=8` (swap in the Red Hat host address). Check available options with `socat -hh | grep multicast`.
- Reverse the roles (send from Pi, receive on Red Hat) to confirm both directions independently of the Go tool.

## Configuration experiments
- Set a higher TTL/hop limit (e.g. 8) via new CLI support so packets survive transit even if an intermediate switch/router decrements TTL.
- Temporarily add the multicast route noted above to bypass the loopback default; if it works, bake it into your network scripts.
- Relax reverse-path filtering (`sysctl net.ipv4.conf.all.rp_filter=2`) in case asymmetric paths are being rejected.
- Check SELinux audit logs: `sudo ausearch -m avc -ts recent` for multicast-related denials.
- Inspect the LAN gear (IGMP snooping tables or multicast filtering) if packets leave the host but never reach the Pi.
- If packets arrive on the Pi per `tcpdump` but the app remains silent, double-check `iptables -L -n` (or `nft list ruleset`) and confirm `net.ipv4.conf.*.rp_filter` is not discarding them.
- As a fallback on multicast-restricted networks, run `./multichat -broadcast -group 192.168.1.255` to deliver messages via subnet broadcast.
