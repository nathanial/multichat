package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	maxUDPPayload = 65507
)

type chatMessage struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Body   string `json:"body"`
	SentAt int64  `json:"sent_at"`
}

func main() {
	group := flag.String("group", "239.42.0.1", "multicast group address (IPv4 or IPv6)")
	port := flag.Int("port", 9999, "UDP port to use for the multicast group")
	nickname := flag.String("name", "", "display name to use in the chat (defaults to your username)")
	ifaceName := flag.String("iface", "", "network interface name to join for multicast traffic (optional)")
	flag.Parse()

	if *port <= 0 || *port > 65535 {
		fmt.Fprintf(os.Stderr, "invalid port: %d\n", *port)
		os.Exit(2)
	}

	groupIP := net.ParseIP(*group)
	if groupIP == nil {
		fmt.Fprintf(os.Stderr, "invalid multicast address: %s\n", *group)
		os.Exit(2)
	}
	if !groupIP.IsMulticast() {
		fmt.Fprintf(os.Stderr, "%s is not a multicast address\n", *group)
		os.Exit(2)
	}

	network := "udp4"
	if groupIP.To4() == nil {
		network = "udp6"
	}

	name := strings.TrimSpace(*nickname)
	if name == "" {
		name = defaultName()
	}

	var joinInterface *net.Interface
	if strings.TrimSpace(*ifaceName) != "" {
		ifi, err := net.InterfaceByName(*ifaceName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to use interface %q: %v\n", *ifaceName, err)
			os.Exit(2)
		}
		joinInterface = ifi
	}

	wantsIPv4 := groupIP.To4() != nil
	multicastAddr := &net.UDPAddr{IP: groupIP, Port: *port}
	if joinInterface != nil && !wantsIPv4 {
		multicastAddr.Zone = joinInterface.Name
	}

	recvConn, err := net.ListenMulticastUDP(network, joinInterface, multicastAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to join multicast group: %v\n", err)
		os.Exit(1)
	}
	defer recvConn.Close()

	_ = recvConn.SetReadBuffer(64 * 1024)

	var localSendAddr *net.UDPAddr
	if joinInterface != nil {
		if addr, err := interfaceLocalAddr(joinInterface, wantsIPv4); err == nil {
			localSendAddr = addr
		} else {
			fmt.Fprintf(os.Stderr, "warning: %v; relying on system default route\n", err)
		}
	}

	sendConn, err := net.DialUDP(network, localSendAddr, multicastAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up sender socket: %v\n", err)
		os.Exit(1)
	}
	defer sendConn.Close()

	clientID := randomID()

	var printMu sync.Mutex

	printMu.Lock()
	fmt.Printf("Joined multicast chat %s:%d over %s as %s\n", multicastAddr.IP, multicastAddr.Port, network, name)
	fmt.Println("Type your messages and press Enter to send. Press Ctrl+C or Ctrl+D to exit.")
	fmt.Print("> ")
	printMu.Unlock()

	incoming := make(chan chatMessage)
	go receiveLoop(recvConn, incoming)

	go func() {
		for msg := range incoming {
			showMessage(msg, clientID, &printMu)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		if !scanner.Scan() {
			break
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			printMu.Lock()
			fmt.Print("> ")
			printMu.Unlock()
			continue
		}

		msg := chatMessage{
			ID:     clientID,
			Name:   name,
			Body:   text,
			SentAt: time.Now().UnixNano(),
		}

		payload, err := json.Marshal(msg)
		if err != nil {
			printMu.Lock()
			fmt.Fprintf(os.Stderr, "failed to encode message: %v\n", err)
			fmt.Print("> ")
			printMu.Unlock()
			continue
		}
		if len(payload) > maxUDPPayload {
			printMu.Lock()
			fmt.Fprintf(os.Stderr, "message too long (max payload %d bytes after encoding)\n", maxUDPPayload)
			fmt.Print("> ")
			printMu.Unlock()
			continue
		}

		if _, err := sendConn.Write(payload); err != nil {
			printMu.Lock()
			fmt.Fprintf(os.Stderr, "failed to send message: %v\n", err)
			fmt.Print("> ")
			printMu.Unlock()
			continue
		}

		printMu.Lock()
		fmt.Print("> ")
		printMu.Unlock()
	}

	if err := scanner.Err(); err != nil {
		printMu.Lock()
		fmt.Fprintf(os.Stderr, "input error: %v\n", err)
		printMu.Unlock()
	}

	printMu.Lock()
	fmt.Println("\nLeaving chat, goodbye!")
	printMu.Unlock()
}

func receiveLoop(conn *net.UDPConn, out chan<- chatMessage) {
	buffer := make([]byte, 64*1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			close(out)
			return
		}
		if n == 0 {
			continue
		}

		var msg chatMessage
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			continue
		}
		msg.Body = strings.TrimRight(msg.Body, "\r\n")
		out <- msg
	}
}

func showMessage(msg chatMessage, selfID string, mu *sync.Mutex) {
	displayName := msg.Name
	if displayName == "" {
		displayName = "anon"
	}
	if msg.ID == selfID {
		displayName = "you"
	}

	ts := time.Now()
	if msg.SentAt != 0 {
		ts = time.Unix(0, msg.SentAt).Local()
	}

	mu.Lock()
	fmt.Printf("\r[%s] <%s> %s\n", ts.Format("15:04:05"), displayName, msg.Body)
	fmt.Print("> ")
	mu.Unlock()
}

func interfaceLocalAddr(iface *net.Interface, wantsIPv4 bool) (*net.UDPAddr, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		default:
			continue
		}
		if ip == nil {
			continue
		}
		if wantsIPv4 {
			ip = ip.To4()
			if ip == nil {
				continue
			}
		} else {
			if ip.To4() != nil {
				continue
			}
			ip = ip.To16()
			if ip == nil {
				continue
			}
		}
		if ip.IsLoopback() {
			continue
		}
		return &net.UDPAddr{IP: ip, Port: 0}, nil
	}
	family := "IPv6"
	if wantsIPv4 {
		family = "IPv4"
	}
	return nil, fmt.Errorf("no %s address found on interface %s", family, iface.Name)
}

func defaultName() string {
	if user := strings.TrimSpace(os.Getenv("USER")); user != "" {
		return user
	}
	if user := strings.TrimSpace(os.Getenv("USERNAME")); user != "" {
		return user
	}
	if host, err := os.Hostname(); err == nil && strings.TrimSpace(host) != "" {
		return host
	}
	return "guest"
}

func randomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
