
package main

import (
	"fmt"
	"os"

	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/tui"
)

func main() {
	state := config.NewState()
	m := tui.NewModel(state)
	// Simulate WindowSizeMsg
	m.Update(nil) // just to be safe, though Update takes tea.Msg
	// We need to set width and height directly but they are unexported?
	// Ah, I cannot access unexported fields from outside the package.
}
