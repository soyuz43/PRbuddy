// test/dce/command_menu/testutils_test.go
package command_menu_test

import (
	"bytes"
	"io"

	"github.com/soyuz43/prbuddy-go/internal/dce"
)

// MockOutputWriter captures output for testing
type MockOutputWriter struct {
	*bytes.Buffer
}

func (m *MockOutputWriter) Write(p []byte) (n int, err error) {
	return m.Buffer.Write(p)
}

// SetOutputForTests allows redirecting output for tests
func SetOutputForTests(w io.Writer) {
	dce.SetOutput(w)
}
