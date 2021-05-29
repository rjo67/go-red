package red

import "testing"

func TestMultipleMarks(t *testing.T) {
	state := resetState([]string{"1", "2", "3", "4", "5"})

	// move to line 2
	cmd, err := ParseCommand("2p", false)
	if err != nil {
		t.Fatalf("error %s", err)
	}
	if _, err = cmd.ProcessCommand(state, nil, false); err != nil {
		t.Fatalf("error %s", err)
	}

	_addMark(t, state, "2", "a")
	assertInt(t, "wrong line number", state.lineNbr, 2)
	assertInt(t, "marks list not correct size", len(state.marks), 1)

	_addMark(t, state, "3", "b")
	assertInt(t, "marks list not correct size", len(state.marks), 2)

	_addMark(t, state, "1", "c")

	// add a mark, reusing the name 'a'
	_addMark(t, state, "1", "a")
	assertInt(t, "marks list not correct size", len(state.marks), 3)

	// add a mark, reusing the name 'b'
	_addMark(t, state, "1", "b")
	assertInt(t, "marks list not correct size", len(state.marks), 3)

	// add a mark, reusing the name 'c'
	_addMark(t, state, "1", "c")
	assertInt(t, "marks list not correct size", len(state.marks), 3)
}

func TestInvalidNameOfMark(t *testing.T) {
	var cmd Command
	var err error
	// marknames must be just one char a-z
	if cmd, err = ParseCommand("2kabc", false); err != nil {
		t.Fatalf("error %s", err)
	}
	state := resetState([]string{"1", "2", "3", "4", "5"})
	cmd.resolveAddress(state)
	if err = cmd.Mark(state); err != errBadMarkname {
		t.Fatalf("expected 'errBadMarkname', got '%v'", err)
	}
}

// various tests with marks, when lines are deleted
func TestLineDeletion(t *testing.T) {
	state := resetState([]string{"1", "2", "3", "4", "5"})
	_addMark(t, state, "2", "a")
	_addMark(t, state, "4", "b")

	// delete line 1 :   a->1, b->3
	_delete(t, state, "1")
	assertInt(t, "mark 'a' not pointing at correct line.", state.marks["a"].lineNbr, 1)
	assertInt(t, "mark 'b' not pointing at correct line.", state.marks["b"].lineNbr, 3)

	// delete line 3 :   a->1
	_delete(t, state, "3")
	assertInt(t, "mark 'a' not pointing at correct line.", state.marks["a"].lineNbr, 1)
	if elem, ok := state.marks["b"]; ok != false {
		t.Fatalf("mark 'b' should not exist, but got: %v", elem)
	}
}

func TestLineMoveFromBelow(t *testing.T) {
	state := resetState([]string{"a", "b", "c", "d", "e", "f"})
	_addMark(t, state, "3", "a")

	// no-op
	_move(t, state, "1", "2")
	assertInt(t, "mark 'a' not pointing at correct line.", state.marks["a"].lineNbr, 3)
	_move(t, state, "1", "3")
	assertInt(t, "mark 'a' not pointing at correct line.", state.marks["a"].lineNbr, 2)

	state = resetState([]string{"a", "b", "c", "d", "e", "f"})
	_addMark(t, state, "3", "a")
	_move(t, state, "2", "4")
	assertInt(t, "mark 'a' not pointing at correct line.", state.marks["a"].lineNbr, 2)

	state = resetState([]string{"a", "b", "c", "d", "e", "f"})
	_addMark(t, state, "3", "a")
	_move(t, state, "1,2", "4")
	assertInt(t, "mark 'a' not pointing at correct line.", state.marks["a"].lineNbr, 1)

	state = resetState([]string{"a", "b", "c", "d", "e", "f"})
	_addMark(t, state, "2", "a")
	_addMark(t, state, "4", "b") // mark on destination line
	_addMark(t, state, "5", "c")
	_move(t, state, "1,2", "4")
	// {"c", "d", "a", "b", "e", "f"})
	assertElementDoesNotExist(t, state.marks, "a")
	assertInt(t, "mark 'b' not pointing at correct line.", state.marks["b"].lineNbr, 2)
	assertInt(t, "mark 'c' not pointing at correct line.", state.marks["c"].lineNbr, 5)
}

func TestLineMoveFromAbove(t *testing.T) {
	state := resetState([]string{"a", "b", "c", "d", "e", "f"})
	_addMark(t, state, "2", "a")

	// no-op
	_move(t, state, "3", "4")
	assertInt(t, "mark 'a' not pointing at correct line.", state.marks["a"].lineNbr, 2)

	_move(t, state, "3", "1")
	assertInt(t, "mark 'a' not pointing at correct line.", state.marks["a"].lineNbr, 3)

	state = resetState([]string{"a", "b", "c", "d", "e", "f"})
	_addMark(t, state, "3", "a")
	_move(t, state, "4,6", "2")
	assertInt(t, "mark 'a' not pointing at correct line.", state.marks["a"].lineNbr, 6)

	state = resetState([]string{"a", "b", "c", "d", "e", "f", "g"})
	_addMark(t, state, "2", "a")
	_addMark(t, state, "3", "b") // mark on destination line
	_addMark(t, state, "5", "c")
	_addMark(t, state, "7", "d") // above moved lines
	_move(t, state, "5,6", "3")
	// {"a", "b", "c", "e", "f", "d"})
	assertInt(t, "mark 'a' not pointing at correct line.", state.marks["a"].lineNbr, 2)
	assertInt(t, "mark 'b' not pointing at correct line.", state.marks["b"].lineNbr, 3)
	assertElementDoesNotExist(t, state.marks, "c")
	assertInt(t, "mark 'd' not pointing at correct line.", state.marks["d"].lineNbr, 7)
}

func assertElementDoesNotExist(t *testing.T, marks map[string]Mark, key string) {
	if elem, ok := marks[key]; ok != false {
		t.Fatalf("mark '%s' should not exist, but got: %v", key, elem)
	}
}

func _addMark(t *testing.T, state *State, addrRange, markName string) {
	var err error
	var cmd Command
	if cmd, err = createCommandAndResolveAddressRange(state, newValidRange(addrRange), commandMark, markName); err != nil {
		t.Fatalf("error parsing command '%s'k%s: %s", addrRange, markName, err.Error())
	}
	if err = cmd.Mark(state); err != nil {
		t.Fatalf("error processing command %v: %s", cmd, err)
	}
}

func _delete(t *testing.T, state *State, addrRange string) {
	var cmd Command
	var err error
	if cmd, err = createCommandAndResolveAddressRange(state, newValidRange(addrRange), commandDelete, ""); err != nil {
		t.Fatalf("error %s", err)
	}
	if err = cmd.Delete(state, true); err != nil {
		t.Fatalf("error %s", err)
	}
}

func _move(t *testing.T, state *State, addrRange, destination string) {
	var cmd Command
	var err error
	if cmd, err = createCommandAndResolveAddressRange(state, newValidRange(addrRange), commandMove, destination); err != nil {
		t.Fatalf("error: %s", err)
	}
	if err = cmd.Move(state); err != nil {
		t.Fatalf("error: %s", err)
	}
}
