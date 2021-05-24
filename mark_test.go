package red

import "testing"

func TestMark(t *testing.T) {
	state := NewState()
	state.Buffer = createListOfLines([]string{"1", "2", "3", "4", "5"})

	// move to line 2
	cmd, err := ParseCommand("2p")
	if err != nil {
		t.Fatalf("error %s", err)
	}
	_, err = cmd.ProcessCommand(state, nil, false)
	if err != nil {
		t.Fatalf("error %s", err)
	}

	_addMark(t, state, "2ka")

	// current address should be unchanged
	if state.lineNbr != 2 {
		t.Fatalf("wrong state.lineNbr! got %d, expected 2", state.lineNbr)
	}
	_checkSizeOfMarkList(t, state, 1)
}

func TestMultipleMarks(t *testing.T) {
	state := NewState()
	state.Buffer = createListOfLines([]string{"1", "2", "3", "4", "5"})

	// move to line 2
	cmd, err := ParseCommand("2p")
	if err != nil {
		t.Fatalf("error %s", err)
	}
	_, err = cmd.ProcessCommand(state, nil, false)
	if err != nil {
		t.Fatalf("error %s", err)
	}

	_addMark(t, state, "2ka")
	_addMark(t, state, "3kb")

	_checkSizeOfMarkList(t, state, 2)

	_addMark(t, state, "1kc")

	// add a mark, reusing the name 'a'
	_addMark(t, state, "1ka")
	_checkSizeOfMarkList(t, state, 3)

	// add a mark, reusing the name 'b'
	_addMark(t, state, "1kb")
	_checkSizeOfMarkList(t, state, 3)

	// add a mark, reusing the name 'c'
	_addMark(t, state, "1kc")
	_checkSizeOfMarkList(t, state, 3)
}

func TestInvalidNameOfMark(t *testing.T) {
	var cmd Command
	var err error
	// marknames must be just one char a-z
	if cmd, err = ParseCommand("2kabc"); err != nil {
		t.Fatalf("error %s", err)
	}
	err = cmd.CmdMark(NewState())
	if err != errBadMarkname {
		t.Fatalf("expected 'errBadMarkname'")
	}
}

func _addMark(t *testing.T, state *State, markCommand string) {
	cmd, err := ParseCommand(markCommand)
	if err != nil {
		t.Fatalf("error parsing command '%s': %s", markCommand, err)
	}
	err = cmd.CmdMark(state)
	if err != nil {
		t.Fatalf("error processing command %v: %s", cmd, err)
	}
}

func _checkSizeOfMarkList(t *testing.T, state *State, expectedSize int) {
	if len(state.marks) != expectedSize {
		t.Fatalf("marks list not correct size! got %d, expected %d", len(state.marks), expectedSize)
	}
}
