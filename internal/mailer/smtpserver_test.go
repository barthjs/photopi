package mailer

import (
	"bufio"
	"crypto/hmac"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// receivedMail is one message a fake SMTP server accepted.
type receivedMail struct {
	From       string
	Recipients []string
	Data       string
	// Username is the user that authenticated before the message, empty without authentication.
	Username string
}

// fakeSMTPServer is a minimal SMTP server on a local port. It speaks just enough of the protocol for a client
// to deliver messages and records what it got. With a password set, it requires CRAM-MD5 authentication,
// which clients accept on a connection without TLS.
type fakeSMTPServer struct {
	listener net.Listener
	// port is the loopback port the server listens on.
	port int
	// password enables authentication and is the password every user must prove.
	password string
	// rejectRecipient answers RCPT TO for this address with a permanent failure.
	rejectRecipient string

	mu       sync.Mutex
	received []receivedMail
}

// newFakeSMTPServer starts a fake SMTP server on a free loopback port.
func newFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()

	var config net.ListenConfig
	listener, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	server := &fakeSMTPServer{
		listener: listener,
		port:     address.Port,
	}
	t.Cleanup(func() { _ = listener.Close() })
	go server.serve()
	return server
}

// messages returns the messages accepted so far.
func (s *fakeSMTPServer) messages() []receivedMail {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]receivedMail(nil), s.received...)
}

// serve accepts connections until the listener closes.
func (s *fakeSMTPServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

// handle answers the SMTP commands of one connection and records the accepted messages.
func (s *fakeSMTPServer) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	reply := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }
	reply("220 fake ESMTP")

	var current receivedMail
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		command := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			if s.password != "" {
				reply("250-fake")
				reply("250 AUTH CRAM-MD5")
			} else {
				reply("250 fake")
			}
		case command == "AUTH CRAM-MD5":
			challenge := "<challenge@fake>"
			reply("334 " + base64.StdEncoding.EncodeToString([]byte(challenge)))
			answer, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			decoded, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(answer))
			user, digest, _ := strings.Cut(string(decoded), " ")
			mac := hmac.New(md5.New, []byte(s.password))
			_, _ = mac.Write([]byte(challenge))
			if digest != hex.EncodeToString(mac.Sum(nil)) {
				reply("535 authentication failed")
				continue
			}
			current.Username = user
			reply("235 authenticated")
		case strings.HasPrefix(command, "MAIL FROM:"):
			current.From = addressOf(line[len("MAIL FROM:"):])
			reply("250 ok")
		case strings.HasPrefix(command, "RCPT TO:"):
			recipient := addressOf(line[len("RCPT TO:"):])
			if recipient == s.rejectRecipient {
				reply("550 no such user")
				continue
			}
			current.Recipients = append(current.Recipients, recipient)
			reply("250 ok")
		case command == "DATA":
			reply("354 send data")
			var data strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if dataLine == ".\r\n" {
					break
				}
				_, _ = data.WriteString(dataLine)
			}
			current.Data = data.String()
			s.mu.Lock()
			s.received = append(s.received, current)
			s.mu.Unlock()
			current = receivedMail{
				Username: current.Username,
			}
			reply("250 queued")
		case command == "QUIT":
			reply("221 bye")
			return
		default:
			// RSET, NOOP, and anything else the client sends between messages
			reply("250 ok")
		}
	}
}

// addressOf extracts the address from a MAIL FROM or RCPT TO argument such as "<a@b.c> SIZE=1".
func addressOf(argument string) string {
	argument = strings.TrimSpace(argument)
	if end := strings.Index(argument, ">"); strings.HasPrefix(argument, "<") && end > 0 {
		return argument[1:end]
	}
	if fields := strings.Fields(argument); len(fields) > 0 {
		return fields[0]
	}
	return argument
}

// unusedPort returns a local port nothing listens on, for connection failures.
func unusedPort(t *testing.T) int {
	t.Helper()

	var config net.ListenConfig
	listener, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	require.NoError(t, listener.Close())
	return address.Port
}
