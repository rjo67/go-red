package red

import (
	"container/list"
	"fmt"
	"regexp"
)

type Line struct {
	Line string
}

/*
State stores the global state.
*/
type State struct {
	// the last line number is accessible via buffer.Len()
	Buffer                *list.List     // the current buffer -- should never be null
	CutBuffer             *list.List     // the cut buffer, set by commands c, d, j, s or y
	dotline               *list.Element  // the current (dot) line -- can be null
	marks                 map[string]int // file marks
	lineNbr               int            // the current line number
	lastSubstRE           *regexp.Regexp // the previous substitution regexp
	lastSubstReplacement  string         // the previous substitution replacement string
	lastSubstSuffixes     string         // the previous substitution suffixes
	lastSearchRE          *regexp.Regexp // the previous search regexp
	undo                  *list.List     // list of commands to undo
	processingUndo        bool           // if currently processing an undo (therefore don't add undo commands)
	changedSinceLastWrite bool           // whether the buffer has been changed since the last write
	ProgramFlags
}

type ProgramFlags struct {
	defaultFilename string // name of the default file
	WindowSize      int    // window size - for scroll command
	Debug           bool   // cmdline flag: debugging activated?
	ShowMemory      bool   // cmdline flag: show memory stats?
	Prompt          string // cmdline flag: the prompt string
	ShowPrompt      bool   // whether to show the prompt
}

/*
Undo stores information about the inverse of the current command, and is stored in the undo list (which is held in State).
 Some commands (e.g. move) require a multi-command undo. This is handled internally using a special command.
*/
type Undo struct {
	cmd         Command    // the command required to undo what has just been changed
	text        *list.List // text which was changed
	originalCmd Command    // for when we implement 'redo'
}

/*
NewState initialises a state structure.
*/
func NewState() *State {
	state := State{}
	state.Buffer = list.New()
	state.CutBuffer = list.New()
	state.marks = make(map[string]int)
	state.undo = list.New()
	state.Prompt = ":" // default prompt

	return &state
}

/*
 Adds an undo command to the list held in the state.
 Does nothing if we're already processing an "undo".
*/
func (state *State) addUndo(start, end int, command string, text *list.List, origCmd Command) {
	if !state.processingUndo {
		startAddr := newAbsoluteAddress(start)
		endAddr := newAbsoluteAddress(end)
		undoCommand := Undo{Command{addrRange: AddressRange{startAddr, endAddr, separatorComma}, cmd: command, restOfCmd: ""}, text, origCmd}
		if state.Debug {
			fmt.Println("added undo:", undoCommand)
		}
		state.undo.PushFront(undoCommand)
	}
}
