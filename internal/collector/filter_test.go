package collector

import (
	"testing"
)

func TestFilterConnections(t *testing.T) {
	conns := []Connection{
		{PID: 1, Process: "proc1", User: "user1", Proto: "tcp", State: "ESTABLISHED", Laddr: "1.1.1.1", Lport: 80, Raddr: "2.2.2.2", Rport: 1234},
		{PID: 2, Process: "proc2", User: "user2", Proto: "udp", State: "LISTEN", Laddr: "3.3.3.3", Lport: 53, Raddr: "*", Rport: 0},
		{PID: 3, Process: "proc1_extra", User: "user1", Proto: "tcp", State: "ESTABLISHED", Laddr: "4.4.4.4", Lport: 443, Raddr: "5.5.5.5", Rport: 5678},
	}

	testCases := []struct {
		name     string
		filters  FilterOptions
		expected int
	}{
		{"No filters", FilterOptions{}, 3},
		{"Filter by proto tcp", FilterOptions{Proto: "tcp"}, 2},
		{"Filter by proto udp", FilterOptions{Proto: "udp"}, 1},
		{"Filter by state", FilterOptions{State: "ESTABLISHED"}, 2},
		{"Filter by pid", FilterOptions{Pid: 2}, 1},
		{"Filter by proc", FilterOptions{Proc: "proc1"}, 2},
		{"Filter by lport", FilterOptions{Lport: 80}, 1},
		{"Filter by rport", FilterOptions{Rport: 1234}, 1},
		{"Filter by user", FilterOptions{User: "user1"}, 2},
		{"Filter by laddr", FilterOptions{Laddr: "1.1.1.1"}, 1},
		{"Filter by raddr", FilterOptions{Raddr: "5.5.5.5"}, 1},
		{"Filter by contains proc", FilterOptions{Contains: "proc2"}, 1},
		{"Filter by contains addr", FilterOptions{Contains: "3.3.3.3"}, 1},
		{"Combined filter", FilterOptions{Proto: "tcp", State: "ESTABLISHED"}, 2},
		{"No match", FilterOptions{Proto: "tcp", State: "LISTEN"}, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filtered := FilterConnections(conns, tc.filters)
			if len(filtered) != tc.expected {
				t.Errorf("Expected %d connections, but got %d", tc.expected, len(filtered))
			}
		})
	}
}
