package main

import (
	"container/list"
	"errors"
	"fmt"
)

var nothingToUndo error = errors.New("Nothing to undo")

/*
 When processing a command, its inverse is stored in the undo list (see State).

 TODO
 The list holds elements of type []Undo, because some commands require a multi-level undo.

*/
type Undo struct {
	cmd         Command    // the command required to undo what has just been changed
	text        *list.List // text which was changed
	originalCmd Command    // for when we implement 'redo'
}

func (cmd Command) CmdUndo(state *State) error {

	if state.undo.Len() == 0 {
		return nothingToUndo
	}

	undoEl := state.undo.Front()
	state.undo.Remove(undoEl)
	undo := undoEl.Value.(Undo)

	// set global flag to indicate we're undoing
	state.processingUndo = true
	fmt.Println(undo.cmd)
	_, err := processCommand(undo.cmd, state, undo.text, false)
	state.processingUndo = false
	return err
}
