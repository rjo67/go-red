package red

import (
	"bytes"
	"fmt"
	"testing"
)

func TestAppend(t *testing.T) {
	data := []struct {
		cmdIdent         string
		addrRange        string
		inputLines       []string
		expectedContents string
		expectedLineNbr  int
	}{
		{commandAppend, "2", []string{"some new lines", "another one"}, "1\n2\nsome new lines\nanother one\n3\n4\n5\n", 4},
		/* append at start */ {commandAppend, "0", []string{"new line 1"}, "new line 1\n1\n2\n3\n4\n5\n", 1},
		/* append at end */ {commandAppend, "5", []string{"6", "7", "8"}, "1\n2\n3\n4\n5\n6\n7\n8\n", 8},
		{commandInsert, "2", []string{"some new lines", "another one"}, "1\nsome new lines\nanother one\n2\n3\n4\n5\n", 3},
		/* insert at start */ {commandInsert, "0", []string{"new line 1"}, "new line 1\n1\n2\n3\n4\n5\n", 1},
		/* insert at end */ {commandInsert, "5", []string{"6", "7", "8"}, "1\n2\n3\n4\n6\n7\n8\n5\n", 7},
	}

	for i, test := range data {
		t.Run(fmt.Sprintf("test %d: >>%s<<", i, test.addrRange), func(t *testing.T) {
			var err error
			var cmd Command
			state := resetState([]string{"1", "2", "3", "4", "5"})
			if cmd, err = createCommandAndResolveAddressRange(state, newValidRange(test.addrRange), test.cmdIdent, ""); err != nil {
				t.Fatalf("error: %s", err)
			}
			if err = cmd.AppendInsert(state, createListOfLines(test.inputLines)); err != nil {
				t.Fatalf("error: %s", err)
			}
			assertInt(t, "wrong state.lineNbr!", state.lineNbr, test.expectedLineNbr)
			assertBufferContents(t, state.Buffer, test.expectedContents)
		})
	}

}

func TestChange(t *testing.T) {
	data := []struct {
		addrRange        string
		inputLines       []string
		expectedContents string
		expectedLineNbr  int
	}{
		{"2,+3" /*start line is 2*/, []string{"some new lines", "another one"}, "1\nsome new lines\nanother one\n", 3},
		{"1", []string{"new line 1"}, "new line 1\n2\n3\n4\n5\n", 1},
		{"5", []string{"6", "7", "8"}, "1\n2\n3\n4\n6\n7\n8\n", 7},
		{"1,$", []string{"6", "7", "8"}, "6\n7\n8\n", 3},
	}

	for i, test := range data {
		t.Run(fmt.Sprintf("test %d: >>%s<<", i, test.addrRange), func(t *testing.T) {
			var err error
			var cmd Command
			state := resetState([]string{"1", "2", "3", "4", "5"})
			state.lineNbr = 2 // necessary for some tests
			if cmd, err = createCommandAndResolveAddressRange(state, newValidRange(test.addrRange), commandChange, ""); err != nil {
				t.Fatalf("error: %s", err)
			}
			if err = cmd.Change(state, createListOfLines(test.inputLines)); err != nil {
				t.Fatalf("error: %s", err)
			}
			assertInt(t, "wrong state.lineNbr!", state.lineNbr, test.expectedLineNbr)
			assertBufferContents(t, state.Buffer, test.expectedContents)
		})
	}
}

func TestDelete(t *testing.T) {
	var err error
	var cmd Command
	state := resetState([]string{"1", "2", "3", "4", "5"})
	if cmd, err = createCommandAndResolveAddressRange(state, newValidRange("2,3"), commandDelete, ""); err != nil {
		t.Fatalf("error %s", err)
	}

	if err = cmd.Delete(state, true); err != nil {
		t.Fatalf("error %s", err)
	}
	assertInt(t, "wrong state.lineNbr!", state.lineNbr, 2)
	assertBufferContents(t, state.Buffer, "1\n4\n5\n")

	// delete whole file
	state = resetState([]string{"1", "2", "3", "4", "5"})
	if cmd, err = createCommandAndResolveAddressRange(state, newValidRange("1, 5"), commandDelete, ""); err != nil {
		t.Fatalf("error %s", err)
	}
	if err = cmd.Delete(state, true); err != nil {
		t.Fatalf("error %s", err)
	}
	assertInt(t, "wrong state.lineNbr!", state.lineNbr, 0)
	assertBufferContents(t, state.Buffer, "")
}

func TestJoin(t *testing.T) {
	var err error
	var cmd Command
	state := resetState([]string{"1", "2", "3", "4", "5"})
	if cmd, err = createCommandAndResolveAddressRange(state, newValidRange("2,3"), commandJoin, ""); err != nil {
		t.Fatalf("error: %s", err)
	}
	if err = cmd.Join(state); err != nil {
		t.Fatalf("error: %s", err)
	}
	assertInt(t, "wrong state.lineNbr!", state.lineNbr, 2)
	assertBufferContents(t, state.Buffer, "1\n2 3\n4\n5\n")

	// ---------- test2: NB here "2j" is a no-op, but in vim 2j is the same as 2,+1j
	state = resetState([]string{"1", "2", "3", "4", "5"})
	state.lineNbr = 2
	if cmd, err = createCommandAndResolveAddressRange(state, newValidRange("2"), commandJoin, ""); err != nil {
		t.Fatalf("error: %s", err)
	}
	if err = cmd.Join(state); err != nil {
		t.Fatalf("error: %s", err)
	}
	assertBufferContents(t, state.Buffer, "1\n2\n3\n4\n5\n")
}

func TestMove(t *testing.T) {
	data := []struct {
		addrRange        string
		destination      string
		expectedContents string
		expectedLineNbr  int
	}{
		{"2,+3" /*start line is 2*/, "0", "2\n3\n4\n5\n1\n", 4}, // move to start of file
		{"1", "2", "2\n1\n3\n4\n5\n", 2},
		{"1,2", "5", "3\n4\n5\n1\n2\n", 5}, // move to after last line
		{"5", "3", "1\n2\n3\n5\n4\n", 4},   // move last line
		{"1,$", "0", "1\n2\n3\n4\n5\n", 5}, //no-op
	}

	for i, test := range data {
		t.Run(fmt.Sprintf("test %d: >>%s<<", i, test.addrRange), func(t *testing.T) {
			var err error
			var cmd Command
			state := resetState([]string{"1", "2", "3", "4", "5"})
			state.lineNbr = 2 // necessary for some tests
			if cmd, err = createCommandAndResolveAddressRange(state, newValidRange(test.addrRange), commandMove, test.destination); err != nil {
				t.Fatalf("error: %s", err)
			}
			if err = cmd.Move(state); err != nil {
				t.Fatalf("error: %s", err)
			}
			assertInt(t, "wrong state.lineNbr!", state.lineNbr, test.expectedLineNbr)
			assertBufferContents(t, state.Buffer, test.expectedContents)
		})
	}
}

func TestPrintRange(t *testing.T) {
	var err error
	var cmd Command
	state := resetState([]string{"1", "2", "3", "4", "5"})
	if cmd, err = createCommandAndResolveAddressRange(state, newValidRange("2, 3"), commandPrint, ""); err != nil {
		t.Fatalf("error %s", err)
	}

	// to capture the output
	var buff bytes.Buffer // implements io.Writer

	if err := _printRange(&buff, cmd, state, false); err != nil {
		t.Fatalf("error %s", err)
	}
	if buff.String() != "2\n3\n" {
		t.Fatalf("2,3p returned '%s'", buff.String())
	}

	buff.Reset()
	if cmd, err = createCommandAndResolveAddressRange(state, newValidRange("1, 4"), commandPrint, ""); err != nil {
		t.Fatalf("error %s", err)
	}
	if err = _printRange(&buff, cmd, state, false); err != nil {
		t.Fatalf("error %s", err)
	}
	if buff.String() != "1\n2\n3\n4\n" {
		t.Fatalf("1,4p returned '%s'", buff.String())
	}

	buff.Reset()
	if cmd, err = createCommandAndResolveAddressRange(state, newValidRange("3,3"), commandPrint, ""); err != nil {
		t.Fatalf("error %s", err)
	}
	if err = _printRange(&buff, cmd, state, false); err != nil {
		t.Fatalf("error %s", err)
	}
	if buff.String() != "3\n" {
		t.Fatalf("3,3p returned '%s'", buff.String())
	}

	buff.Reset()
	// currently at line 3
	if cmd, err = ParseCommand("+1"); err != nil {
		t.Fatalf("error %s", err)
	}
	if err = cmd.resolveAddress(state); err != nil {
		t.Fatalf("error %s", err)
	}
	if err = _printRange(&buff, cmd, state, false); err != nil {
		t.Fatalf("error %s", err)
	}
	if buff.String() != "4\n" {
		t.Fatalf("+1 returned '%s'", buff.String())
	}
}

func TestTransfer(t *testing.T) {
	data := []struct {
		addrRange        string
		destination      string
		expectedContents string
		expectedLineNbr  int
	}{
		{"2,+3" /*start line is 2*/, "0", "2\n3\n4\n5\n1\n2\n3\n4\n5\n", 4}, // transfer to start of file
		{"1", "2", "1\n2\n1\n3\n4\n5\n", 3},
		{"1,2", "5", "1\n2\n3\n4\n5\n1\n2\n", 7},          // transfer to after last line
		{"5", "3", "1\n2\n3\n5\n4\n5\n", 4},               // transfer last line
		{"1,$", "0", "1\n2\n3\n4\n5\n1\n2\n3\n4\n5\n", 5}, // transfers whole file
	}

	for i, test := range data {
		t.Run(fmt.Sprintf("test %d: >>%s<<", i, test.addrRange), func(t *testing.T) {
			var err error
			var cmd Command
			state := resetState([]string{"1", "2", "3", "4", "5"})
			state.lineNbr = 2 // necessary for some tests
			if cmd, err = createCommandAndResolveAddressRange(state, newValidRange(test.addrRange), commandTransfer, test.destination); err != nil {
				t.Fatalf("error: %s", err)
			}
			if err = cmd.Transfer(state); err != nil {
				t.Fatalf("error: %s", err)
			}
			assertInt(t, "wrong state.lineNbr!", state.lineNbr, test.expectedLineNbr)
			assertBufferContents(t, state.Buffer, test.expectedContents)
		})
	}
}

func TestMoveToLine(t *testing.T) {
	data := []string{"first", "second", "3", "", "5"}
	state := resetState(data)

	for i, expected := range data {
		moveToLine(i+1, state)
		assertInt(t, "wrong state.lineNbr!", i+1, state.lineNbr)
		if state.dotline.Value.(Line).Line != expected+"\n" {
			t.Fatalf("bad data element %d, expected '%s' but got '%s'", i, expected, state.dotline.Value.(Line).Line)
		}
	}
}
