package red

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"strings"
	"testing"
)

/*
Creates a *list.List with elements corresponding to the given slice elements.
*/
func createListOfLines(lines []string) *list.List {
	listOfLines := list.New()
	for _, line := range lines {
		listOfLines.PushBack(Line{line + "\n"})
	}
	return listOfLines
}

/*
Creates a Command object from the given parameters AND resolves the address range with the given state.
*/
func createCommandAndResolveAddressRange(state *State, ra AddressRange, cmdIdent, restOfCmd string) (Command, error) {
	cmd := Command{addrRange: ra, cmd: cmdIdent, restOfCmd: restOfCmd}
	err := cmd.resolveAddress(state)
	return cmd, err
}

/*
Checks the buffer contents against the expected string, returns non-nil if they didn't match.
*/
func checkBufferContents(buffer *list.List, expected string) error {
	var err error
	var buff bytes.Buffer               // implements io.Writer
	var writer = bufio.NewWriter(&buff) // -> bufio

	if _, err = WriteWriter(writer, buffer.Front(), 1, buffer.Len()); err != nil {
		return fmt.Errorf("error: %w", err)
	}
	if buff.String() != expected {
		return fmt.Errorf("buffers did not match.\nBuffer1: %s\nBuffer2: %s", strings.ReplaceAll(buff.String(), "\n", "\\n"), strings.ReplaceAll(expected, "\n", "\\n"))
	}
	return nil
}

func resetState(buffer []string) *State {
	state := NewState()
	state.Buffer = createListOfLines(buffer)
	return state
}

func assertInt(t *testing.T, text string, val, expected int) {
	if val != expected {
		t.Fatalf("assert failed: %s got %d, expected %d", text, val, expected)
	}
}

func assertBufferContents(t *testing.T, buffer *list.List, expected string) {
	if err := checkBufferContents(buffer, expected); err != nil {
		t.Fatalf("error: %s", err)
	}
}
