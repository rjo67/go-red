package main

import (
	"fmt"
)

/**
 * Stores information about a line.
 * The line number is not stored, this is implicit.
 */
type Line struct {
	line string
}
func (l Line) String() string {
	return fmt.Sprintf("line of length %d", len(l.line))
}

func main() {
	fmt.Println("hello")
}
