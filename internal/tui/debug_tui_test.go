package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/st1lson/glitch/internal/config"
)

func TestDebugLayout(t *testing.T) {
	state := config.NewManager(config.Config{})
	m := NewModel(state)
	m.width = 100
	m.height = 30
	out := m.View()

	debugOut := strings.ReplaceAll(out, " ", ".")

	err := os.WriteFile("debug_layout.txt", []byte(debugOut), 0644)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(out, "\n")
	var report strings.Builder
	for i, line := range lines {

		fmt.Fprintf(&report, "Line %d: bytes=%d chars=%d\n", i+1, len(line), len([]rune(line)))
	}
	os.WriteFile("debug_report.txt", []byte(report.String()), 0644)
}
