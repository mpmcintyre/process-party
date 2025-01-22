package pp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockWriter implements io.Writer for testing
type mockWriter struct {
	written []byte
	err     error
}

// Utility write function to test writer
func (m *mockWriter) Write(p []byte) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	m.written = append(m.written, p...)
	return len(p), nil
}

// Ensure that empty lines are cleared
func TestEmptyMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		emptyExpexted bool
	}{
		{"empty string", "", true},
		{"single space", " ", true},
		{"multiple spaces", "   ", true},
		{"tabs", "\t\t", true},
		{"spaces and tabs", "  \t  ", true},
		{"non-empty string", "hello", false},
		{"string with spaces", "  hello  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := emptyMessage(tt.input)
			assert.Equal(t, tt.emptyExpexted, result)
		})
	}
}

// Ensures that the writer has the correct lengths
func TestCustomWriterBasicOutput(t *testing.T) {
	t.Parallel()

	mock := &mockWriter{}
	process := &Process{
		Name:   "test",
		Prefix: "TEST",
	}
	writer := &customWriter{
		w:        mock,
		process:  process,
		severity: "info",
	}

	message := "hello world"
	n, err := writer.Write([]byte(message))

	assert.NoError(t, err)
	assert.Equal(t, len(message), n)

	// Get the color escape sequences
	coloredPrefix := process.GetFgColour()("[TEST]")

	// Test the actual colored output
	expectedOutput := fmt.Sprintf("%s %s\r\n", coloredPrefix, message)
	assert.Equal(t, expectedOutput, string(mock.written))
}

// Ensure that the colour ouput is provided on output string
func TestCustomWriterColorOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		color    ColourCode
		severity string
	}{
		{"yellow color", ColourCmdYellow, "info"},
		{"blue color", ColourCmdBlue, "info"},
		{"red color", ColourCmdRed, "info"},
		{"error message", ColourCmdWhite, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWriter{}
			process := &Process{
				Name:   "test",
				Prefix: "TEST",
				Color:  tt.color,
			}

			writer := &customWriter{
				w:        mock,
				process:  process,
				severity: tt.severity,
			}

			_, err := writer.Write([]byte("test message"))
			assert.NoError(t, err)
			assert.NotEmpty(t, mock.written)
			// Note: We can't easily test the actual colors as they're ANSI codes,
			// but we can verify the message structure
			assert.Contains(t, string(mock.written), "[TEST]")
			assert.Contains(t, string(mock.written), "test message")
		})
	}
}

// Ensure that the prefix is formatted properly
func TestPrefixFormatting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		prefix     string
		displayPid bool
		pid        string
		expected   string
	}{
		{"basic prefix", "TEST", false, "", "[TEST]"},
		{"prefix with pid", "TEST", true, "123", "[TEST-123]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWriter{}
			process := &Process{
				Prefix:     tt.prefix,
				DisplayPid: tt.displayPid,
				Pid:        tt.pid,
			}
			writer := &customWriter{
				w:       mock,
				process: process,
			}
			writer.createPrefix()
			assert.Contains(t, writer.prefix, tt.expected)
		})
	}
}

// Ensure that newlines contiain prefixes
func TestLineSeparation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		separateLines bool
		expectedLines int
		containsEmpty bool
	}{
		{
			name:          "multiple lines separated",
			input:         "line1\nline2\nline3",
			expectedLines: 3,
			containsEmpty: false,
		},
		{
			name:          "empty lines filtered",
			input:         "line1\n\nline2\n\nline3",
			expectedLines: 5,
			containsEmpty: false,
		},
		{
			name:          "single line preserved",
			input:         "single line\n",
			expectedLines: 2,
			containsEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWriter{}
			process := &Process{
				Prefix: "TEST",
			}
			writer := &customWriter{
				w:       mock,
				process: process,
			}

			_, err := writer.Write([]byte(tt.input))
			assert.NoError(t, err)

			lines := strings.Split(string(mock.written), "\n")
			// Remove last empty line from splitting
			if lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}

			assert.Equal(t, tt.expectedLines, len(lines))
			if !tt.containsEmpty {
				for _, line := range lines {
					assert.NotEmpty(t, strings.TrimSpace(line))
				}
			}
		})
	}
}

// Ensure the timestamp works
func TestTimestampFeature(t *testing.T) {
	t.Parallel()

	mock := &mockWriter{}
	process := &Process{
		Prefix:        "TEST",
		ShowTimestamp: true,
	}
	writer := &customWriter{
		w:       mock,
		process: process,
	}

	_, err := writer.Write([]byte("test message"))
	assert.NoError(t, err)

	output := string(mock.written)
	// Verify timestamp format HH:MM:SS:MS is present
	assert.Regexp(t, `\d{2}:\d{2}:\d{2}:\d{3}`, output)
}

// Ensure that no output is allowed when silent
func TestSilentMode(t *testing.T) {
	t.Parallel()

	mock := &mockWriter{}
	process := &Process{
		Prefix: "TEST",
		Silent: true,
	}
	writer := &customWriter{
		w:       mock,
		process: process,
	}

	n, err := writer.Write([]byte("test message"))
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, mock.written)
}

// Test unnamed test

func TestUnnamed(t *testing.T) {
	t.Parallel()

	mock := &mockWriter{}
	process := &Process{
		Prefix: "",
	}
	writer := &customWriter{
		w:       mock,
		process: process,
	}

	n, err := writer.Write([]byte("test message"))
	assert.NoError(t, err)
	assert.Equal(t, len("test message"), n)
	assert.Equal(t, "test message\r\n", string(mock.written))
}

// Tests writer error handling
func TestErrorHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		writerError error
		shouldError bool
	}{
		{
			name:        "writer error",
			writerError: assert.AnError,
			shouldError: true,
		},
		{
			name:        "successful write",
			writerError: nil,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWriter{err: tt.writerError}
			process := &Process{
				Prefix: "TEST"}
			writer := &customWriter{
				w:       mock,
				process: process,
			}

			_, err := writer.Write([]byte("test message"))
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
