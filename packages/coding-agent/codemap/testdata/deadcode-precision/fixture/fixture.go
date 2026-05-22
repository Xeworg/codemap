package fixture

import "fmt"

// ExportedHelper is part of the public API surface.
// It should be classified uncertain even with no local call edges.
func ExportedHelper() { fmt.Println("helper") }

// privateUnused is never called within this package.
// Lowercase name so it is NOT public API — should be classified unused.
func privateUnused() { fmt.Println("unused") }

// privateUsed is called by main.
func privateUsed() { fmt.Println("used") }

func init() {
	// init functions are runtime entrypoints.
}

// T is a type with a method that has no external callers.
type T struct{}

// Method has no inbound call edges from outside this fixture.
// Methods are uncertain pending owner/context analysis.
func (t T) Method() {}
