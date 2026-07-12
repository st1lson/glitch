
package tui

import (
	"fmt"
	"os"
	"testing"

	"github.com/st1lson/glitch/internal/config"
)

func TestViewLayout3(t *testing.T) {
	state := config.NewState(config.Config{})
	m := NewModel(state)
	m.width = 100
	m.height = 30
	out := m.View()
	err := os.WriteFile("test_layout3.txt", []byte(out), 0644)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Layout written to test_layout3.txt")
}
