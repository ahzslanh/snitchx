//go:build windows

package collector

import (
	"encoding/binary"
	"os"
	"testing"
	"time"
	"unsafe"
)

func TestWinPort(t *testing.T) {
	tests := []struct {
		in   uint32
		want int
	}{
		{portBytes(80), 80},
		{portBytes(443), 443},
		{portBytes(8080), 8080},
		{portBytes(0), 0},
		{portBytes(65535), 65535},
	}

	for _, tt := range tests {
		if got := winPort(tt.in); got != tt.want {
			t.Errorf("winPort(%#x) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestTCPStateString(t *testing.T) {
	tests := []struct {
		state uint32
		want  string
	}{
		{winTCPStateClosed, "CLOSED"},
		{winTCPStateListen, "LISTEN"},
		{winTCPStateSynSent, "SYN_SENT"},
		{winTCPStateSynRecv, "SYN_RECV"},
		{winTCPStateEstablished, "ESTABLISHED"},
		{winTCPStateFinWait1, "FIN_WAIT1"},
		{winTCPStateFinWait2, "FIN_WAIT2"},
		{winTCPStateCloseWait, "CLOSE_WAIT"},
		{winTCPStateClosing, "CLOSING"},
		{winTCPStateLastAck, "LAST_ACK"},
		{winTCPStateTimeWait, "TIME_WAIT"},
		{winTCPStateDeleteTCB, "DELETE_TCB"},
		{0, ""},
		{99, ""},
	}

	for _, tt := range tests {
		if got := tcpStateString(tt.state); got != tt.want {
			t.Errorf("tcpStateString(%d) = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestIPv4String(t *testing.T) {
	tests := []struct {
		in   uint32
		want string
	}{
		{0x0100007F, "127.0.0.1"},
		{0x08080808, "8.8.8.8"},
		{0, "0.0.0.0"},
		{0x00000001, "1.0.0.0"},
	}

	for _, tt := range tests {
		if got := ipv4String(tt.in); got != tt.want {
			t.Errorf("ipv4String(%#x) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeWinAddr(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"0.0.0.0", "*"},
		{"::", "*"},
		{"127.0.0.1", "127.0.0.1"},
		{"::1", "::1"},
	}

	for _, tt := range tests {
		if got := normalizeWinAddr(tt.in); got != tt.want {
			t.Errorf("normalizeWinAddr(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDecodeTCP4Table(t *testing.T) {
	now := time.Now()
	buf := encodeTable([]mibTCPRow{{
		State:      winTCPStateEstablished,
		LocalAddr:  0x0100007F,
		LocalPort:  portBytes(8080),
		RemoteAddr: 0x08080808,
		RemotePort: portBytes(443),
		OwningPID:  1234,
	}})

	lookup := func(pid int) winProcInfo {
		if pid != 1234 {
			t.Fatalf("unexpected pid lookup: %d", pid)
		}
		return winProcInfo{name: "test.exe", user: `DOMAIN\user`}
	}

	conns := decodeTCP4Table(buf, now, lookup)
	if len(conns) != 1 {
		t.Fatalf("got %d connections, want 1", len(conns))
	}

	c := conns[0]
	if c.Proto != "tcp" || c.IPVersion != "IPv4" {
		t.Errorf("proto/version = %q/%q, want tcp/IPv4", c.Proto, c.IPVersion)
	}
	if c.State != "ESTABLISHED" {
		t.Errorf("state = %q, want ESTABLISHED", c.State)
	}
	if c.Laddr != "127.0.0.1" || c.Lport != 8080 {
		t.Errorf("local = %s:%d, want 127.0.0.1:8080", c.Laddr, c.Lport)
	}
	if c.Raddr != "8.8.8.8" || c.Rport != 443 {
		t.Errorf("remote = %s:%d, want 8.8.8.8:443", c.Raddr, c.Rport)
	}
	if c.PID != 1234 || c.Process != "test.exe" || c.User != `DOMAIN\user` {
		t.Errorf("process = %d/%q/%q, want 1234/test.exe/DOMAIN\\user", c.PID, c.Process, c.User)
	}
	if c.TS.IsZero() {
		t.Error("expected TS to be set")
	}
}

func TestDecodeTCP6Table(t *testing.T) {
	var loopback [16]byte
	loopback[15] = 1

	buf := encodeTable([]mibTCP6Row{{
		LocalAddr: loopback,
		LocalPort: portBytes(443),
		State:     winTCPStateListen,
		OwningPID: 99,
	}})

	conns := decodeTCP6Table(buf, time.Now(), func(int) winProcInfo { return winProcInfo{} })
	if len(conns) != 1 {
		t.Fatalf("got %d connections, want 1", len(conns))
	}

	c := conns[0]
	if c.Proto != "tcp6" || c.IPVersion != "IPv6" {
		t.Errorf("proto/version = %q/%q, want tcp6/IPv6", c.Proto, c.IPVersion)
	}
	if c.State != "LISTEN" {
		t.Errorf("state = %q, want LISTEN", c.State)
	}
	if c.Laddr != "::1" || c.Lport != 443 {
		t.Errorf("local = %s:%d, want [::1]:443", c.Laddr, c.Lport)
	}
	if c.Raddr != "*" || c.Rport != 0 {
		t.Errorf("remote = %s:%d, want *:0", c.Raddr, c.Rport)
	}
}

func TestDecodeUDP4Table(t *testing.T) {
	buf := encodeTable([]mibUDPRow{{
		LocalAddr: 0,
		LocalPort: portBytes(5353),
		OwningPID: 77,
	}})

	conns := decodeUDP4Table(buf, time.Now(), func(int) winProcInfo { return winProcInfo{name: "dns.exe"} })
	if len(conns) != 1 {
		t.Fatalf("got %d connections, want 1", len(conns))
	}

	c := conns[0]
	if c.Proto != "udp" || c.IPVersion != "IPv4" {
		t.Errorf("proto/version = %q/%q, want udp/IPv4", c.Proto, c.IPVersion)
	}
	if c.State != "LISTEN" {
		t.Errorf("state = %q, want LISTEN", c.State)
	}
	if c.Laddr != "*" || c.Lport != 5353 {
		t.Errorf("local = %s:%d, want *:5353", c.Laddr, c.Lport)
	}
	if c.Raddr != "*" || c.Rport != 0 {
		t.Errorf("remote = %s:%d, want *:0", c.Raddr, c.Rport)
	}
	if c.PID != 77 || c.Process != "dns.exe" {
		t.Errorf("process = %d/%q, want 77/dns.exe", c.PID, c.Process)
	}
}

func TestDecodeEmptyTable(t *testing.T) {
	for _, buf := range [][]byte{nil, {}, {0, 0, 0, 0}, {1, 0, 0}} {
		if conns := decodeTCP4Table(buf, time.Now(), func(int) winProcInfo { return winProcInfo{} }); len(conns) != 0 {
			t.Errorf("expected no connections for malformed buffer %v, got %d", buf, len(conns))
		}
	}
}

func TestLookupWinProcessSelf(t *testing.T) {
	info := lookupWinProcess(uint32(os.Getpid()))
	if info.name == "" {
		t.Error("expected process name for the test binary")
	}
}

func TestGetConnectionsWindows(t *testing.T) {
	conns, err := GetConnections()
	if err != nil {
		t.Fatalf("GetConnections() failed: %v", err)
	}

	t.Logf("enumerated %d connections", len(conns))

	for _, c := range conns {
		if c.Proto == "" {
			t.Errorf("connection has empty proto: %+v", c)
		}
		if c.IPVersion != "IPv4" && c.IPVersion != "IPv6" {
			t.Errorf("unexpected ip version %q", c.IPVersion)
		}
		if c.Lport < 0 || c.Lport > 65535 || c.Rport < 0 || c.Rport > 65535 {
			t.Errorf("port out of range: %s:%d -> %s:%d", c.Laddr, c.Lport, c.Raddr, c.Rport)
		}
		if c.Laddr == "" {
			t.Errorf("connection has empty local address: %+v", c)
		}
	}
}

// portBytes encodes a port the way the IP helper API stores it: network byte
// order in the low 16 bits of a DWORD.
func portBytes(port int) uint32 {
	value := uint16(port)
	return uint32(value>>8 | value<<8)
}

// encodeTable builds a MIB_*TABLE_OWNER_PID buffer with a 4-byte entry count
// followed by the rows.
func encodeTable[T any](rows []T) []byte {
	rowSize := int(unsafe.Sizeof(*new(T)))
	buf := make([]byte, 4+len(rows)*rowSize)
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(rows)))

	dst := unsafe.Slice((*T)(unsafe.Pointer(&buf[4])), len(rows))
	copy(dst, rows)
	return buf
}

func TestDedupeConnections(t *testing.T) {
	base := Connection{
		Proto:     "udp",
		IPVersion: "IPv4",
		State:     "LISTEN",
		Laddr:     "*",
		Lport:     3702,
		Raddr:     "*",
		PID:       4812,
	}
	other := base
	other.Lport = 3703
	other6 := base
	other6.Proto = "udp6"
	other6.IPVersion = "IPv6"

	conns := []Connection{base, base, other, other6}
	got := dedupeConnections(conns)

	if len(got) != 3 {
		t.Fatalf("dedupeConnections returned %d rows, want 3: %+v", len(got), got)
	}
	if got[0].Lport != 3702 || got[1].Lport != 3703 || got[2].Proto != "udp6" {
		t.Errorf("unexpected dedupe order/content: %+v", got)
	}
}
