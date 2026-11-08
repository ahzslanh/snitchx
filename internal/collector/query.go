package collector

// Query combines filtering, sorting, and limiting into a single operation
type Query struct {
	Filter FilterOptions
	Sort   SortOptions
	Limit  int
}

// NewQuery creates a query with sensible defaults
func NewQuery() *Query {
	return &Query{
		Filter: FilterOptions{},
		Sort:   SortOptions{Field: SortByLport, Direction: SortAsc},
		Limit:  0,
	}
}

// WithFilter sets the filter options
func (q *Query) WithFilter(f FilterOptions) *Query {
	q.Filter = f
	return q
}

// WithSort sets the sort options
func (q *Query) WithSort(s SortOptions) *Query {
	q.Sort = s
	return q
}

// WithSortString parses and sets sort options from a string like "pid:desc"
func (q *Query) WithSortString(s string) *Query {
	q.Sort = ParseSortOptions(s)
	return q
}

// WithLimit sets the maximum number of results
func (q *Query) WithLimit(n int) *Query {
	q.Limit = n
	return q
}

// Proto filters by protocol
func (q *Query) Proto(proto string) *Query {
	q.Filter.Proto = proto
	return q
}

// State filters by connection state
func (q *Query) State(state string) *Query {
	q.Filter.State = state
	return q
}

// Process filters by process name (substring match)
func (q *Query) Process(proc string) *Query {
	q.Filter.Proc = proc
	return q
}

// PID filters by process ID
func (q *Query) PID(pid int) *Query {
	q.Filter.Pid = pid
	return q
}

// LocalPort filters by local port
func (q *Query) LocalPort(port int) *Query {
	q.Filter.Lport = port
	return q
}

// RemotePort filters by remote port
func (q *Query) RemotePort(port int) *Query {
	q.Filter.Rport = port
	return q
}

// IPv4Only filters to only IPv4 connections
func (q *Query) IPv4Only() *Query {
	q.Filter.IPv4 = true
	q.Filter.IPv6 = false
	return q
}

// IPv6Only filters to only IPv6 connections
func (q *Query) IPv6Only() *Query {
	q.Filter.IPv4 = false
	q.Filter.IPv6 = true
	return q
}

// Listening filters to only listening sockets
func (q *Query) Listening() *Query {
	q.Filter.State = "LISTEN"
	return q
}

// Established filters to only established connections
func (q *Query) Established() *Query {
	q.Filter.State = "ESTABLISHED"
	return q
}

// Contains filters by substring in process, local addr, or remote addr
func (q *Query) Contains(s string) *Query {
	q.Filter.Contains = s
	return q
}

// Execute runs the query and returns results
func (q *Query) Execute() ([]Connection, error) {
	conns, err := GetConnections()
	if err != nil {
		return nil, err
	}

	return q.Apply(conns), nil
}

// Apply applies the query to a slice of connections
func (q *Query) Apply(conns []Connection) []Connection {
	result := FilterConnections(conns, q.Filter)
	SortConnections(result, q.Sort)

	if q.Limit > 0 && len(result) > q.Limit {
		result = result[:q.Limit]
	}

	return result
}

// common pre-built queries

// ListeningTCP returns a query for TCP listeners
func ListeningTCP() *Query {
	return NewQuery().Proto("tcp").Listening()
}

// ListeningAll returns a query for all listeners
func ListeningAll() *Query {
	return NewQuery().Listening()
}

// EstablishedTCP returns a query for established TCP connections
func EstablishedTCP() *Query {
	return NewQuery().Proto("tcp").Established()
}

// ByProcess returns a query filtered by process name
func ByProcess(name string) *Query {
	return NewQuery().Process(name)
}

// ByPort returns a query filtered by local port
func ByPort(port int) *Query {
	return NewQuery().LocalPort(port)
}
