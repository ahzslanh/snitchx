package collector

import (
	"testing"
)

func TestQueryBuilder(t *testing.T) {
	t.Run("fluent builder pattern", func(t *testing.T) {
		q := NewQuery().
			Proto("tcp").
			State("LISTEN").
			WithLimit(10)

		if q.Filter.Proto != "tcp" {
			t.Errorf("expected proto tcp, got %s", q.Filter.Proto)
		}
		if q.Filter.State != "LISTEN" {
			t.Errorf("expected state LISTEN, got %s", q.Filter.State)
		}
		if q.Limit != 10 {
			t.Errorf("expected limit 10, got %d", q.Limit)
		}
	})

	t.Run("convenience methods", func(t *testing.T) {
		q := NewQuery().Listening()
		if q.Filter.State != "LISTEN" {
			t.Errorf("Listening() should set state to LISTEN")
		}

		q = NewQuery().Established()
		if q.Filter.State != "ESTABLISHED" {
			t.Errorf("Established() should set state to ESTABLISHED")
		}

		q = NewQuery().IPv4Only()
		if q.Filter.IPv4 != true || q.Filter.IPv6 != false {
			t.Error("IPv4Only() should set IPv4=true, IPv6=false")
		}

		q = NewQuery().IPv6Only()
		if q.Filter.IPv4 != false || q.Filter.IPv6 != true {
			t.Error("IPv6Only() should set IPv4=false, IPv6=true")
		}
	})

	t.Run("sort options", func(t *testing.T) {
		q := NewQuery().WithSortString("pid:desc")

		if q.Sort.Field != SortByPID {
			t.Errorf("expected sort by pid, got %v", q.Sort.Field)
		}
		if q.Sort.Direction != SortDesc {
			t.Errorf("expected sort desc, got %v", q.Sort.Direction)
		}
	})
}

func TestQueryApply(t *testing.T) {
	conns := []Connection{
		{PID: 1, Process: "nginx", Proto: "tcp", State: "LISTEN", Lport: 80},
		{PID: 2, Process: "nginx", Proto: "tcp", State: "ESTABLISHED", Lport: 80},
		{PID: 3, Process: "sshd", Proto: "tcp", State: "LISTEN", Lport: 22},
		{PID: 4, Process: "postgres", Proto: "tcp", State: "LISTEN", Lport: 5432},
		{PID: 5, Process: "dnsmasq", Proto: "udp", State: "", Lport: 53},
	}

	t.Run("filter by state", func(t *testing.T) {
		q := NewQuery().Listening()
		result := q.Apply(conns)

		if len(result) != 3 {
			t.Errorf("expected 3 listening connections, got %d", len(result))
		}
	})

	t.Run("filter by proto", func(t *testing.T) {
		q := NewQuery().Proto("udp")
		result := q.Apply(conns)

		if len(result) != 1 {
			t.Errorf("expected 1 udp connection, got %d", len(result))
		}
		if result[0].Process != "dnsmasq" {
			t.Errorf("expected dnsmasq, got %s", result[0].Process)
		}
	})

	t.Run("filter and sort", func(t *testing.T) {
		q := NewQuery().Listening().WithSortString("lport:asc")
		result := q.Apply(conns)

		if len(result) != 3 {
			t.Fatalf("expected 3, got %d", len(result))
		}
		if result[0].Lport != 22 {
			t.Errorf("expected port 22 first, got %d", result[0].Lport)
		}
	})

	t.Run("filter sort and limit", func(t *testing.T) {
		q := NewQuery().Proto("tcp").WithSortString("lport:asc").WithLimit(2)
		result := q.Apply(conns)

		if len(result) != 2 {
			t.Errorf("expected 2 (limit), got %d", len(result))
		}
	})

	t.Run("process filter substring", func(t *testing.T) {
		q := NewQuery().Process("nginx")
		result := q.Apply(conns)

		if len(result) != 2 {
			t.Errorf("expected 2 nginx connections, got %d", len(result))
		}
	})

	t.Run("contains filter", func(t *testing.T) {
		q := NewQuery().Contains("post")
		result := q.Apply(conns)

		if len(result) != 1 || result[0].Process != "postgres" {
			t.Errorf("expected postgres, got %v", result)
		}
	})
}

func TestPrebuiltQueries(t *testing.T) {
	t.Run("ListeningTCP", func(t *testing.T) {
		q := ListeningTCP()
		if q.Filter.Proto != "tcp" || q.Filter.State != "LISTEN" {
			t.Error("ListeningTCP should filter tcp + LISTEN")
		}
	})

	t.Run("ListeningAll", func(t *testing.T) {
		q := ListeningAll()
		if q.Filter.State != "LISTEN" {
			t.Error("ListeningAll should filter LISTEN state")
		}
	})

	t.Run("EstablishedTCP", func(t *testing.T) {
		q := EstablishedTCP()
		if q.Filter.Proto != "tcp" || q.Filter.State != "ESTABLISHED" {
			t.Error("EstablishedTCP should filter tcp + ESTABLISHED")
		}
	})

	t.Run("ByProcess", func(t *testing.T) {
		q := ByProcess("nginx")
		if q.Filter.Proc != "nginx" {
			t.Error("ByProcess should set process filter")
		}
	})

	t.Run("ByPort", func(t *testing.T) {
		q := ByPort(8080)
		if q.Filter.Lport != 8080 {
			t.Error("ByPort should set lport filter")
		}
	})
}
