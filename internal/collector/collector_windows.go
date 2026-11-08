//go:build windows

package collector

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	winAFInet  = 2
	winAFInet6 = 23

	// TCP_TABLE_CLASS values from iphlpapi.h
	winTCPTableOwnerPIDAll = 5
	// UDP_TABLE_CLASS values from iphlpapi.h
	winUDPTableOwnerPID = 1

	winErrorInsufficientBuffer = 122
	winErrorNoData             = 232

	// MIB_TCP_STATE values from iprtrmib.h
	winTCPStateClosed      = 1
	winTCPStateListen      = 2
	winTCPStateSynSent     = 3
	winTCPStateSynRecv     = 4
	winTCPStateEstablished = 5
	winTCPStateFinWait1    = 6
	winTCPStateFinWait2    = 7
	winTCPStateCloseWait   = 8
	winTCPStateClosing     = 9
	winTCPStateLastAck     = 10
	winTCPStateTimeWait    = 11
	winTCPStateDeleteTCB   = 12
)

var (
	iphlpapiDLL             = windows.NewLazySystemDLL("iphlpapi.dll")
	getExtendedTCPTableProc = iphlpapiDLL.NewProc("GetExtendedTcpTable")
	getExtendedUDPTableProc = iphlpapiDLL.NewProc("GetExtendedUdpTable")
)

// MIB_TCPROW_OWNER_PID
type mibTCPRow struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPID  uint32
}

// MIB_TCP6ROW_OWNER_PID
type mibTCP6Row struct {
	LocalAddr     [16]byte
	LocalScopeID  uint32
	LocalPort     uint32
	RemoteAddr    [16]byte
	RemoteScopeID uint32
	RemotePort    uint32
	State         uint32
	OwningPID     uint32
}

// MIB_UDPROW_OWNER_PID
type mibUDPRow struct {
	LocalAddr uint32
	LocalPort uint32
	OwningPID uint32
}

// MIB_UDP6ROW_OWNER_PID
type mibUDP6Row struct {
	LocalAddr    [16]byte
	LocalScopeID uint32
	LocalPort    uint32
	OwningPID    uint32
}

type winProcInfo struct {
	name    string
	cmdline string
	cwd     string
	user    string
	uid     int
}

type winProcCache struct {
	mu sync.Mutex
	m  map[int]winProcInfo
}

// DefaultCollector implements the Collector interface using the Windows IP
// helper API (iphlpapi.dll) to enumerate TCP/UDP sockets and their owning PIDs.
type DefaultCollector struct{}

// GetConnections fetches all IPv4 and IPv6 TCP/UDP connections on Windows.
func (dc *DefaultCollector) GetConnections() ([]Connection, error) {
	now := time.Now()
	cache := &winProcCache{m: make(map[int]winProcInfo)}
	lookup := cache.lookup

	tables := []struct {
		name   string
		proc   *windows.LazyProc
		af     uint32
		class  uint32
		decode func([]byte, time.Time, func(int) winProcInfo) []Connection
	}{
		{"tcp4", getExtendedTCPTableProc, winAFInet, winTCPTableOwnerPIDAll, decodeTCP4Table},
		{"tcp6", getExtendedTCPTableProc, winAFInet6, winTCPTableOwnerPIDAll, decodeTCP6Table},
		{"udp4", getExtendedUDPTableProc, winAFInet, winUDPTableOwnerPID, decodeUDP4Table},
		{"udp6", getExtendedUDPTableProc, winAFInet6, winUDPTableOwnerPID, decodeUDP6Table},
	}

	var (
		connections []Connection
		errs        []error
		succeeded   int
	)

	for _, table := range tables {
		buf, err := queryTable(table.proc, table.af, table.class)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", table.name, err))
			continue
		}
		succeeded++
		connections = append(connections, table.decode(buf, now, lookup)...)
	}

	if succeeded == 0 {
		return nil, fmt.Errorf("failed to enumerate windows connections: %w", errors.Join(errs...))
	}

	return dedupeConnections(connections), nil
}

// dedupeConnections removes identical rows. Windows can return the same UDP
// endpoint once per network compartment, which otherwise shows duplicate rows.
func dedupeConnections(connections []Connection) []Connection {
	if len(connections) < 2 {
		return connections
	}

	type key struct {
		proto, ipVersion, state string
		laddr, raddr            string
		lport, rport            int
		pid                     int
	}

	seen := make(map[key]struct{}, len(connections))
	deduped := connections[:0]
	for _, conn := range connections {
		k := key{
			proto:     conn.Proto,
			ipVersion: conn.IPVersion,
			state:     conn.State,
			laddr:     conn.Laddr,
			raddr:     conn.Raddr,
			lport:     conn.Lport,
			rport:     conn.Rport,
			pid:       conn.PID,
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		deduped = append(deduped, conn)
	}
	return deduped
}

// GetAllConnections returns network connections on Windows.
// Unix domain sockets are not part of the Windows IP helper tables.
func GetAllConnections() ([]Connection, error) {
	return GetConnections()
}

// queryTable allocates a buffer and calls GetExtendedTcpTable/GetExtendedUdpTable.
// The first call with a nil table is used to discover the required size.
func queryTable(proc *windows.LazyProc, af, class uint32) ([]byte, error) {
	var size uint32

	ret, _, _ := proc.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		uintptr(af),
		uintptr(class),
		0,
	)
	if ret == winErrorNoData {
		return nil, nil
	}
	if ret != 0 && ret != winErrorInsufficientBuffer {
		return nil, fmt.Errorf("size query failed with error %d", ret)
	}
	if size < 4 {
		return nil, nil
	}

	for attempt := 0; attempt < 4; attempt++ {
		buf := make([]byte, size)
		ret, _, _ = proc.Call(
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&size)),
			0,
			uintptr(af),
			uintptr(class),
			0,
		)
		if ret == 0 || ret == winErrorNoData {
			return buf, nil
		}
		if ret != winErrorInsufficientBuffer {
			return nil, fmt.Errorf("enumerate failed with error %d", ret)
		}
		// size was updated by the API; retry with the larger buffer.
	}

	return nil, fmt.Errorf("table size kept changing during enumeration")
}

func decodeTCP4Table(buf []byte, now time.Time, lookup func(int) winProcInfo) []Connection {
	count, rows, ok := tableRows[mibTCPRow](buf)
	if !ok {
		return nil
	}

	connections := make([]Connection, 0, count)
	for i := 0; i < count; i++ {
		row := rows[i]
		pid := int(row.OwningPID)
		connections = append(connections, newConnection(
			now,
			"tcp",
			"IPv4",
			tcpStateString(row.State),
			ipv4String(row.LocalAddr),
			winPort(row.LocalPort),
			ipv4String(row.RemoteAddr),
			winPort(row.RemotePort),
			pid,
			lookup(pid),
		))
	}
	return connections
}

func decodeTCP6Table(buf []byte, now time.Time, lookup func(int) winProcInfo) []Connection {
	count, rows, ok := tableRows[mibTCP6Row](buf)
	if !ok {
		return nil
	}

	connections := make([]Connection, 0, count)
	for i := 0; i < count; i++ {
		row := rows[i]
		pid := int(row.OwningPID)
		connections = append(connections, newConnection(
			now,
			"tcp6",
			"IPv6",
			tcpStateString(row.State),
			ipv6String(row.LocalAddr),
			winPort(row.LocalPort),
			ipv6String(row.RemoteAddr),
			winPort(row.RemotePort),
			pid,
			lookup(pid),
		))
	}
	return connections
}

func decodeUDP4Table(buf []byte, now time.Time, lookup func(int) winProcInfo) []Connection {
	count, rows, ok := tableRows[mibUDPRow](buf)
	if !ok {
		return nil
	}

	connections := make([]Connection, 0, count)
	for i := 0; i < count; i++ {
		row := rows[i]
		pid := int(row.OwningPID)
		// The UDP owner table carries no remote address or state.
		connections = append(connections, newConnection(
			now,
			"udp",
			"IPv4",
			"LISTEN",
			ipv4String(row.LocalAddr),
			winPort(row.LocalPort),
			"*",
			0,
			pid,
			lookup(pid),
		))
	}
	return connections
}

func decodeUDP6Table(buf []byte, now time.Time, lookup func(int) winProcInfo) []Connection {
	count, rows, ok := tableRows[mibUDP6Row](buf)
	if !ok {
		return nil
	}

	connections := make([]Connection, 0, count)
	for i := 0; i < count; i++ {
		row := rows[i]
		pid := int(row.OwningPID)
		connections = append(connections, newConnection(
			now,
			"udp6",
			"IPv6",
			"LISTEN",
			ipv6String(row.LocalAddr),
			winPort(row.LocalPort),
			"*",
			0,
			pid,
			lookup(pid),
		))
	}
	return connections
}

// tableRows validates the header and returns the entry count plus a typed view
// over the row array. The returned slice aliases buf.
func tableRows[T any](buf []byte) (int, []T, bool) {
	if len(buf) < 4 {
		return 0, nil, false
	}

	count := int(binary.LittleEndian.Uint32(buf[:4]))
	if count <= 0 {
		return 0, nil, false
	}

	rowSize := int(unsafe.Sizeof(*new(T)))
	if rowSize <= 0 || len(buf) < 4+count*rowSize {
		return 0, nil, false
	}

	rows := unsafe.Slice((*T)(unsafe.Pointer(&buf[4])), count)
	return count, rows, true
}

func newConnection(
	now time.Time,
	proto, ipVersion, state string,
	laddr string,
	lport int,
	raddr string,
	rport int,
	pid int,
	info winProcInfo,
) Connection {
	laddr = normalizeWinAddr(laddr)
	raddr = normalizeWinAddr(raddr)

	return Connection{
		TS:        now,
		PID:       pid,
		Process:   info.name,
		Cmdline:   info.cmdline,
		Cwd:       info.cwd,
		User:      info.user,
		UID:       info.uid,
		Proto:     proto,
		IPVersion: ipVersion,
		State:     state,
		Laddr:     laddr,
		Lport:     lport,
		Raddr:     raddr,
		Rport:     rport,
		Interface: guessNetworkInterface(laddr),
	}
}

func (c *winProcCache) lookup(pid int) winProcInfo {
	if pid <= 0 {
		return winProcInfo{}
	}

	c.mu.Lock()
	info, exists := c.m[pid]
	c.mu.Unlock()
	if exists {
		return info
	}

	info = lookupWinProcess(uint32(pid))

	c.mu.Lock()
	c.m[pid] = info
	c.mu.Unlock()

	return info
}

func lookupWinProcess(pid uint32) winProcInfo {
	// These processes cannot be opened, but are useful to display by name.
	switch pid {
	case 0:
		return winProcInfo{name: "System Idle Process"}
	case 4:
		return winProcInfo{name: "System"}
	}

	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return winProcInfo{}
	}
	defer windows.CloseHandle(handle)

	var info winProcInfo

	buf := make([]uint16, 32*1024)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err == nil && size > 0 {
		info.name = filepath.Base(windows.UTF16ToString(buf[:size]))
	}

	var token windows.Token
	if err := windows.OpenProcessToken(handle, windows.TOKEN_QUERY, &token); err == nil {
		if tokenUser, err := token.GetTokenUser(); err == nil && tokenUser.User.Sid != nil {
			account, domain, _, lookupErr := tokenUser.User.Sid.LookupAccount("")
			if lookupErr == nil {
				if domain != "" {
					info.user = domain + `\` + account
				} else {
					info.user = account
				}
			}
		}
		_ = token.Close()
	}

	return info
}

func winPort(port uint32) int {
	// The port is stored in network byte order in the low 16 bits.
	value := uint16(port)
	return int(value>>8 | value<<8)
}

func ipv4String(addr uint32) string {
	return net.IPv4(byte(addr), byte(addr>>8), byte(addr>>16), byte(addr>>24)).String()
}

func ipv6String(addr [16]byte) string {
	ip := net.IP(addr[:])
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.String()
}

func normalizeWinAddr(addr string) string {
	if addr == "0.0.0.0" || addr == "::" {
		return "*"
	}
	return addr
}

func tcpStateString(state uint32) string {
	switch state {
	case winTCPStateClosed:
		return "CLOSED"
	case winTCPStateListen:
		return "LISTEN"
	case winTCPStateSynSent:
		return "SYN_SENT"
	case winTCPStateSynRecv:
		return "SYN_RECV"
	case winTCPStateEstablished:
		return "ESTABLISHED"
	case winTCPStateFinWait1:
		return "FIN_WAIT1"
	case winTCPStateFinWait2:
		return "FIN_WAIT2"
	case winTCPStateCloseWait:
		return "CLOSE_WAIT"
	case winTCPStateClosing:
		return "CLOSING"
	case winTCPStateLastAck:
		return "LAST_ACK"
	case winTCPStateTimeWait:
		return "TIME_WAIT"
	case winTCPStateDeleteTCB:
		return "DELETE_TCB"
	default:
		return ""
	}
}
