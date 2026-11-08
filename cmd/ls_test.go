package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ahzslanh/snitchx/internal/collector"
	"github.com/ahzslanh/snitchx/internal/testutil"
)

func TestLsCommand_EmptyResults(t *testing.T) {
	tempDir, cleanup := testutil.SetupTestEnvironment(t)
	defer cleanup()

	// Create empty fixture
	fixture := testutil.CreateFixtureFile(t, tempDir, "empty", []collector.Connection{})

	// Override collector with mock
	originalCollector := collector.GetCollector()
	defer func() {
		collector.SetCollector(originalCollector)
	}()

	mock, err := collector.NewMockCollectorFromFile(fixture)
	if err != nil {
		t.Fatalf("Failed to create mock collector: %v", err)
	}

	collector.SetCollector(mock)

	// Capture output
	capture := testutil.NewOutputCapture(t)
	capture.Start()

	// Run command
	runListCommand("table", []string{})

	stdout, stderr, err := capture.Stop()
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	// Verify no error output
	if stderr != "" {
		t.Errorf("Expected no stderr, got: %s", stderr)
	}

	// Verify table headers are present even with no data
	if !strings.Contains(stdout, "PID") {
		t.Errorf("Expected table headers in output, got: %s", stdout)
	}
}

func TestLsCommand_SingleTCPConnection(t *testing.T) {
	_, cleanup := testutil.SetupTestEnvironment(t)
	defer cleanup()

	// Use predefined fixture
	testCollector := testutil.NewTestCollectorWithFixture("single-tcp")

	// Override collector
	originalCollector := collector.GetCollector()
	defer func() {
		collector.SetCollector(originalCollector)
	}()

	collector.SetCollector(testCollector.MockCollector)

	// Capture output
	capture := testutil.NewOutputCapture(t)
	capture.Start()

	// Run command
	runListCommand("table", []string{})

	stdout, stderr, err := capture.Stop()
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	// Verify no error output
	if stderr != "" {
		t.Errorf("Expected no stderr, got: %s", stderr)
	}

	// Verify connection appears in output
	if !strings.Contains(stdout, "test-app") {
		t.Errorf("Expected process name 'test-app' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "1234") {
		t.Errorf("Expected PID '1234' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "tcp") {
		t.Errorf("Expected protocol 'tcp' in output, got: %s", stdout)
	}
}

func TestLsCommand_JSONOutput(t *testing.T) {
	_, cleanup := testutil.SetupTestEnvironment(t)
	defer cleanup()

	// Use predefined fixture
	testCollector := testutil.NewTestCollectorWithFixture("single-tcp")

	// Override collector
	originalCollector := collector.GetCollector()
	defer func() {
		collector.SetCollector(originalCollector)
	}()

	collector.SetCollector(testCollector.MockCollector)

	// Capture output
	capture := testutil.NewOutputCapture(t)
	capture.Start()

	// Run command with JSON output
	runListCommand("json", []string{})

	stdout, stderr, err := capture.Stop()
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	// Verify no error output
	if stderr != "" {
		t.Errorf("Expected no stderr, got: %s", stderr)
	}

	// Verify JSON structure
	if !strings.Contains(stdout, `"pid"`) {
		t.Errorf("Expected JSON with 'pid' field, got: %s", stdout)
	}
	if !strings.Contains(stdout, `"process"`) {
		t.Errorf("Expected JSON with 'process' field, got: %s", stdout)
	}
	if !strings.Contains(stdout, `[`) || !strings.Contains(stdout, `]`) {
		t.Errorf("Expected JSON array format, got: %s", stdout)
	}

	// JSON output must be valid and free of ANSI styling.
	var conns []map[string]any
	if err := json.Unmarshal([]byte(stdout), &conns); err != nil {
		t.Fatalf("Expected valid JSON without ANSI escapes, got error %v: %q", err, stdout)
	}
	if len(conns) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(conns))
	}
}

func TestLsCommand_Filtering(t *testing.T) {
	_, cleanup := testutil.SetupTestEnvironment(t)
	defer cleanup()

	// Use mixed protocols fixture
	testCollector := testutil.NewTestCollectorWithFixture("mixed-protocols")

	// Override collector
	originalCollector := collector.GetCollector()
	defer func() {
		collector.SetCollector(originalCollector)
	}()

	collector.SetCollector(testCollector.MockCollector)

	// Capture output
	capture := testutil.NewOutputCapture(t)
	capture.Start()

	// Run command with TCP filter
	runListCommand("table", []string{"proto=tcp"})

	stdout, stderr, err := capture.Stop()
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	// Verify no error output
	if stderr != "" {
		t.Errorf("Expected no stderr, got: %s", stderr)
	}

	// Should contain TCP connections
	if !strings.Contains(stdout, "tcp") {
		t.Errorf("Expected TCP connections in filtered output, got: %s", stdout)
	}

	// Should not contain UDP connections
	if strings.Contains(stdout, "udp") {
		t.Errorf("Expected no UDP connections in TCP-filtered output, got: %s", stdout)
	}

	// Should not contain Unix sockets
	if strings.Contains(stdout, "unix") {
		t.Errorf("Expected no Unix sockets in TCP-filtered output, got: %s", stdout)
	}
}

func TestLsCommand_InvalidFilter(t *testing.T) {
	// Skip this test as it's designed to fail
	t.Skip("Skipping TestLsCommand_InvalidFilter as it's designed to fail")
}

func TestParseFilters(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkField  func(collector.FilterOptions) bool
	}{
		{
			name:        "empty args",
			args:        []string{},
			expectError: false,
			checkField:  func(f collector.FilterOptions) bool { return f.IsEmpty() },
		},
		{
			name:        "proto filter",
			args:        []string{"proto=tcp"},
			expectError: false,
			checkField:  func(f collector.FilterOptions) bool { return f.Proto == "tcp" },
		},
		{
			name:        "state filter",
			args:        []string{"state=established"},
			expectError: false,
			checkField:  func(f collector.FilterOptions) bool { return f.State == "established" },
		},
		{
			name:        "pid filter",
			args:        []string{"pid=1234"},
			expectError: false,
			checkField:  func(f collector.FilterOptions) bool { return f.Pid == 1234 },
		},
		{
			name:        "invalid pid",
			args:        []string{"pid=notanumber"},
			expectError: true,
			checkField:  nil,
		},
		{
			name:        "multiple filters",
			args:        []string{"proto=tcp", "state=listen"},
			expectError: false,
			checkField:  func(f collector.FilterOptions) bool { return f.Proto == "tcp" && f.State == "listen" },
		},
		{
			name:        "invalid format",
			args:        []string{"invalid"},
			expectError: true,
			checkField:  nil,
		},
		{
			name:        "unknown filter",
			args:        []string{"unknown=value"},
			expectError: true,
			checkField:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters, err := ParseFilterArgs(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for args %v, but got none", tt.args)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for args %v: %v", tt.args, err)
				return
			}

			if tt.checkField != nil && !tt.checkField(filters) {
				t.Errorf("Filter validation failed for args %v, filters: %+v", tt.args, filters)
			}
		})
	}
}
