package red

import (
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ---- constants for the available commands
const commandAppend string = "a"
const commandChange string = "c"
const commandDelete string = "d"
const CommandEdit string = "e"
const CommandEditUnconditionally string = "E"
const CommandFilename string = "f"
const commandGlobal string = "g"
const commandGlobalInteractive string = "G"
const CommandHelp string = "h" // a startling departure from the ed range of commands ...
const commandInsert string = "i"
const commandJoin string = "j"
const commandMark string = "k"
const commandList string = "l" // print suffix
const commandMove string = "m"
const commandNumber string = "n" // print suffix
const commandPrint string = "p"  // print suffix
const CommandPrompt string = "P"
const CommandQuit string = "q"
const CommandQuitUnconditionally string = "Q"
const commandRead string = "r"
const commandSubstitute string = "s"
const commandTransfer string = "t"
const CommandUndo string = "u"
const commandInverseGlobal string = "v"
const commandInverseGlobalInteractive string = "V"
const commandWrite string = "w"
const commandWriteAppend string = "W"
const commandPut string = "x"
const commandYank string = "y"
const commandScroll string = "z"
const commandComment string = "#"
const commandLinenumber string = "="

// this is an internal command to undo the 'move' command (which requires two steps)
const internalCommandUndoMove string = ")"

// and this is an internal command to undo the 'subst' command (which is 1..n 'change' commands)
const internalCommandUndoSubst string = "("

// returned when an empty line was entered
const commandNoCommand string = ""

const unsavedChanges string = "buffer has unsaved changes"

var errInvalidWindowSize error = errors.New("invalid window size")
var errMissingFilename error = errors.New("filename missing and no default set")
var errNotAllowedInGlobalCommand error = errors.New("command cannot be used within 'g'/'v'")
var errNothingToUndo error = errors.New("Nothing to undo")
var errUnrecognisedCommand error = errors.New("unrecognised command")

var justNumberRE = regexp.MustCompile(`^\s*(\d+)\s*$`)
var commandLineRE = regexp.MustCompile("(.*?)([acdeEfgGhijklmnpPqQrstuvVwWxyz#=])(.*)")

/*
Command stores the command which has been parsed from user input
*/
type Command struct {
	AddrRange AddressRange
	Cmd       string
	restOfCmd string
}

/*
ParseCommand parses a command from the given string.
*/
func ParseCommand(str string) (cmd Command, err error) {
	if strings.TrimSpace(str) == "" {
		// newline alone == +1p
		addrRange, err := newRange("+1")
		if err != nil {
			return Command{}, err
		} else {
			return Command{addrRange, commandPrint, ""}, nil
		}
	}
	// check for a number <n> --> equivalent to <n>p
	matches := justNumberRE.FindStringSubmatch(str)
	if matches != nil {
		addrString := strings.TrimSpace(matches[1])
		addrRange, err := newRange(addrString)
		if err != nil {
			return Command{}, err
		} else {
			return Command{addrRange, commandPrint, ""}, nil
		}
	}
	matches = commandLineRE.FindStringSubmatch(str)
	if matches != nil {
		addrString := strings.TrimSpace(matches[1])
		cmdString := matches[2]
		restOfCmd := matches[3]

		//fmt.Printf("addr: '%s', cmd: '%s', rest: '%s'\n", addrString, cmdString, restOfCmd)
		addrRange, err := newRange(addrString)
		if err != nil {
			return Command{}, err
		} else {
			return Command{addrRange, cmdString, restOfCmd}, nil
		}
	} else {
		return Command{}, errUnrecognisedCommand
	}
}

/*
CmdAppendInsert appends text to the buffer after the addressed line.
 or
inserts text in the buffer before the addressed line.

 The address '0' (zero) is valid for this command;
   append: it places the entered text at the beginning of the buffer.
   insert: it is equivalent to address '1'.

 The current address is set to the address of the last line entered or,
 if there were none, to the addressed line.

 If the parameter "inputLines" is specified, this input will be used.
 Otherwise the user must input the required lines, terminated by ".".
*/
func (cmd Command) CmdAppendInsert(state *State, inputLines *list.List) error {
	// calc required line (checks if this is a valid line)
	lineNbr, err := cmd.AddrRange.start.calculateActualLineNumber(state)
	if err != nil {
		return err
	}

	var newLines *list.List
	var nbrLinesEntered int
	if inputLines != nil {
		newLines = inputLines
		nbrLinesEntered = inputLines.Len()
	} else {
		newLines, nbrLinesEntered, err = readInputLines()
	}
	if nbrLinesEntered == 0 {
		return nil
	}

	state.changedSinceLastWrite = true

	// special case: append line 0
	// or insert with line 0 or 1

	if (cmd.Cmd == commandAppend && lineNbr == 0) || (cmd.Cmd == commandInsert && lineNbr <= 1) {
		state.Buffer.PushFrontList(newLines)
		moveToLine(nbrLinesEntered, state)
		if !state.processingUndo {
			state.addUndo(1, nbrLinesEntered, commandDelete, newLines, cmd)
		}
	} else {
		var startAddrForUndo, endAddrForUndo int
		// an "insert" at line <n> is the same as an append at line <n-1>
		if cmd.Cmd == commandInsert {
			startAddrForUndo = lineNbr
			endAddrForUndo = lineNbr + nbrLinesEntered - 1
			lineNbr--
		} else { // append
			startAddrForUndo = lineNbr + 1
			endAddrForUndo = lineNbr + nbrLinesEntered
		}
		appendLines(lineNbr, state, newLines)

		state.addUndo(startAddrForUndo, endAddrForUndo, commandDelete, newLines, cmd)
	}

	return nil
}

/*
CmdChange changes lines in the buffer.

 The addressed lines are deleted from the buffer, and text is inserted in their place.

 The current address is set to the address of the last line entered or, if there were none,
 to the new address of the line after the last line deleted;
 if the lines deleted were originally at the end of the buffer,
 the current address is set to the address of the new last line;
 if no lines remain in the buffer, the current address is set to zero.
*/
func (cmd Command) CmdChange(state *State, inputLines *list.List) error {

	startLineNbr, err := cmd.AddrRange.start.calculateActualLineNumber(state)
	if err != nil {
		return err
	}
	if startLineNbr == 0 {
		return errInvalidLine
	}

	var newLines *list.List
	var nbrLinesEntered int
	if inputLines != nil {
		newLines = inputLines
		nbrLinesEntered = inputLines.Len()
	} else {
		// get the input, abort if empty
		newLines, nbrLinesEntered, err = readInputLines()
	}
	if nbrLinesEntered == 0 {
		return nil
	}

	// delete the lines
	deleteCmd := Command{cmd.AddrRange, commandDelete, cmd.restOfCmd}
	deleteCmd.CmdDelete(state, false)
	// what's deleted is stored in the cutBuffer

	var atEOF bool

	// adjust the starting lineNbr if we've deleted at the end of the file
	if startLineNbr > state.Buffer.Len() {
		startLineNbr = state.Buffer.Len()
		atEOF = true
	}

	state.changedSinceLastWrite = true

	if atEOF {
		appendLines(startLineNbr, state, newLines)
		// "change" is its own inverse
		state.addUndo(startLineNbr+1, startLineNbr+newLines.Len(), commandChange, state.CutBuffer, cmd)
	} else {
		appendLines(startLineNbr-1, state, newLines)
		state.addUndo(startLineNbr, startLineNbr+newLines.Len()-1, commandChange, state.CutBuffer, cmd)
	}

	return nil
}

/*
CmdComment processes comments.
*/
func (cmd Command) CmdComment(state *State) error {
	// does nothing
	return nil
}

/*
CmdDelete deletes the addressed lines from the buffer.

 The current address is set to the new address of the line after the last line deleted;
 if the lines deleted were originally at the end of the buffer,
   the current address is set to the address of the new last line;
 if no lines remain in the buffer, the current address is set to zero.

 Deleted lines are stored in the state.CutBuffer.

 If addUndo is true, an undo command will be stored in state.undo.
 (This will be affected by the value of state.processingUndo)
*/
func (cmd Command) CmdDelete(state *State, addUndo bool) error {
	startLineNbr, endLineNbr, err := cmd.AddrRange.calculateStartAndEndLineNumbers(state)
	if err != nil {
		return err
	}
	if startLineNbr == 0 {
		return errInvalidLine
	}

	tempBuffer := deleteLines(startLineNbr, endLineNbr, state)

	if tempBuffer.Len() == 0 {
		return nil
	}

	state.CutBuffer = tempBuffer
	state.changedSinceLastWrite = true
	bufferLen := state.Buffer.Len()

	// inverse of delete m..n  ist insert at m
	if addUndo {
		// special case: we've deleted the last line
		if startLineNbr > bufferLen {
			// undo of $d is $-1,a
			state.addUndo(endOfFile, endOfFile, commandAppend, tempBuffer, cmd)
		} else {
			state.addUndo(startLineNbr, startLineNbr, commandInsert, tempBuffer, cmd)
		}
	}

	// set up line nbr and dotline

	newLineNbr := startLineNbr
	if bufferLen == 0 {
		state.dotline = nil
		state.lineNbr = 0
	} else {
		if newLineNbr > bufferLen {
			newLineNbr = bufferLen
		}
		moveToLine(newLineNbr, state)
	}
	return nil
}

/*
CmdEdit edits file, and sets the default filename.
  If file is not specified, then the default filename is used.
  Any lines in the buffer are deleted before the new file is read.
  The current address is set to the address of the last line in the buffer.
  Resets undo buffer.
*/
func (cmd Command) CmdEdit(state *State) error {
	filename, err := getFilename(strings.TrimSpace(cmd.restOfCmd), state, true)
	if err != nil {
		return err
	}
	nbrBytesRead, listOfLines, err := ReadFile(filename)
	if err != nil {
		return err
	}
	fmt.Printf("%dL, %dC\n", listOfLines.Len(), nbrBytesRead)
	state.Buffer = listOfLines
	state.changedSinceLastWrite = false
	state.undo = list.New()
	moveToLine(state.Buffer.Len(), state)
	return nil
}

/*
CmdHelp displays a list of the available commands.
*/
func (cmd Command) CmdHelp(state *State) error {
	fmt.Println()
	fmt.Println(" ", commandAppend, "Appends text after the addressed line.")
	fmt.Println(" ", commandChange, "Changes lines in the buffer.")
	fmt.Println(" ", commandDelete, "Deletes the addressed lines from the buffer.")
	fmt.Println(" ", CommandEdit, "Edits file, and sets the default filename.")
	fmt.Println(" ", CommandEditUnconditionally, "Edits file regardless of any changes in current buffer.")
	fmt.Println(" ", CommandFilename, "Sets the default filename.")
	fmt.Println(" ", commandGlobal, "Executes the command-list for all matching lines.")
	fmt.Println(" ", commandGlobalInteractive, "Interactive 'global'.")
	fmt.Println(" ", CommandHelp, "Displays this help.")
	fmt.Println(" ", commandInsert, "Inserts text before the addressed line.")
	fmt.Println(" ", commandJoin, "Joins the addressed lines, replacing them by a single line containing the joined text.")
	fmt.Println(" ", commandMark, "Marks the current line.")
	fmt.Println(" ", commandList, "Display the addressed lines.")
	fmt.Println(" ", commandMove, "Moves lines in the buffer.")
	fmt.Println(" ", commandNumber, "Displays the addressed lines with line numbers.")
	fmt.Println(" ", commandPrint, "Prints the addressed lines.")
	fmt.Println(" ", CommandPrompt, "Sets the prompt.")
	fmt.Println(" ", CommandQuit, "Quits the editor.")
	fmt.Println(" ", CommandQuitUnconditionally, "Quits the editor without saving changes.")
	fmt.Println(" ", commandRead, "Reads file and appends it after the addressed line.")
	fmt.Println(" ", commandSubstitute, "Replaces text in the addressed lines matching a regular expression.")
	fmt.Println(" ", commandTransfer, "Copies (transfers) the addressed lines to after the right-hand destination address.")
	fmt.Println(" ", CommandUndo, "Undoes the previous command.")
	fmt.Println(" ", commandInverseGlobal, "As 'global' but acts on all lines NOT matching the regex.")
	fmt.Println(" ", commandInverseGlobalInteractive, "Interactive 'inverse-global'.")
	fmt.Println(" ", commandWrite, "Writes the addressed lines to a file.")
	fmt.Println(" ", commandWriteAppend, "Appends the addressed lines to a file.")
	fmt.Println(" ", commandPut, "Copies (puts) the contents of the cut-buffer to after the addressed line.")
	fmt.Println(" ", commandYank, "Copies (yanks) the addressed lines to the cut-buffer.")
	fmt.Println(" ", commandScroll, "Scrolls n lines starting at the addressed line.")
	fmt.Println(" ", commandComment, "Comment -- rest of line will be ignored.")
	fmt.Println(" ", commandLinenumber, "Prints the line number of the addressed line.")
	fmt.Println()

	return nil
}

/*
CmdJoin joins the addressed lines, replacing them by a single line containing their joined text.

 If only one address is given, this command does nothing.

 If lines are joined, the current address is set to the address of the joined line.
 Else, the current address is unchanged.

 Calls internally CmdChange, which is where the undo is handled.
*/
func (cmd Command) CmdJoin(state *State) error {
	var startLineNbr, endLineNbr int
	var err error

	if !cmd.AddrRange.IsAddressRangeSpecified() {
		return nil
	} else {
		if startLineNbr, endLineNbr, err = cmd.AddrRange.calculateStartAndEndLineNumbers(state); err != nil {
			return err
		}
	}
	var sb strings.Builder
	joinFn := func(lineNbr int, el *list.Element, state *State) {
		line := el.Value.(Line).Line
		// replace the newline with a space
		sb.WriteString(strings.Replace(line, "\n", " ", 1))
	}
	iterateLines(startLineNbr, endLineNbr, state, joinFn)
	// add newline again
	sb.WriteString("\n")

	changeCommand := Command{cmd.AddrRange, commandChange, cmd.restOfCmd}

	newLines := list.New()
	newLines.PushBack(Line{sb.String()})
	changeCommand.CmdChange(state, newLines)

	return nil
}

/*
CmdLinenumber prints the line number of the addressed line.

 The current address is unchanged.
*/
func (cmd Command) CmdLinenumber(state *State) error {
	startLineNbr, err := cmd.AddrRange.start.calculateActualLineNumber(state)
	if err != nil {
		return err
	}
	fmt.Println(startLineNbr)
	return nil
}

/*
CmdMove moves lines in the buffer.

 The addressed lines are moved to after the right-hand destination address.
 The destination address '0' (zero) is valid for this command;
    it moves the addressed lines to the beginning of the buffer.

 It is an error if the destination address falls within the range of moved lines.

 The current address is set to the new address of the last line moved.

 Undo is handled by storing the special command 'internalCommandUndoMove'.
*/
func (cmd Command) CmdMove(state *State) error {
	// default is current line (for both start/end, and dest)
	startLineNbr, endLineNbr, err := cmd.AddrRange.getAddressRange(state)
	if err != nil {
		return err
	}
	var destLineNbr int
	// default is current line for destination
	destStr := strings.TrimSpace(cmd.restOfCmd)
	if destStr == "" {
		destLineNbr = state.lineNbr
	} else {
		var destLine Address
		if destLine, err = newAddress(destStr); err != nil {
			return errInvalidDestinationAddress
		}
		if destLineNbr, err = destLine.calculateActualLineNumber(state); err != nil {
			return err
		}
	}
	if startLineNbr == 0 || destLineNbr > state.Buffer.Len() {
		return errInvalidDestinationAddress
	}
	// it is an error if the destination address falls within the range of moved lines
	if destLineNbr >= startLineNbr && destLineNbr < endLineNbr {
		return errInvalidDestinationAddress
	}

	// delete the lines
	tempBuffer := deleteLines(startLineNbr, endLineNbr, state)
	if tempBuffer.Len() == 0 {
		return nil
	}

	// adjust destination line number if it has been affected by the delete
	if destLineNbr >= startLineNbr {
		destLineNbr -= (endLineNbr - startLineNbr + 1)
	}

	appendLines(destLineNbr, state, tempBuffer)
	state.addUndo(destLineNbr+1, destLineNbr+tempBuffer.Len(), internalCommandUndoMove, tempBuffer, cmd)
	state.changedSinceLastWrite = true
	return nil
}

/*
CmdPrint prints the addressed lines. The current address is set to the address of the last line printed.

 For command "n": Precedes each line by its line number and a <tab>.

 The current address is set to the address of the last line printed.
*/
func (cmd Command) CmdPrint(state *State) error {
	// no address specified defaults to .
	if !cmd.AddrRange.IsAddressRangeSpecified() {
		startAddr := Address{state.lineNbr, 0}
		endAddr := Address{state.lineNbr, 0}
		cmd.AddrRange = AddressRange{startAddr, endAddr}
	}
	return _printRange(os.Stdout, cmd, state, cmd.Cmd == commandNumber)
}

/*
CmdPut copies (puts) the contents of the cut buffer to after the addressed line.
 The current address is set to the address of the last line copied.
*/
func (cmd Command) CmdPut(state *State) error {
	var startLineNbr int
	var err error
	// default is append at current line
	if !cmd.AddrRange.IsAddressRangeSpecified() {
		startLineNbr = state.lineNbr
	} else {
		if startLineNbr, err = cmd.AddrRange.start.calculateActualLineNumber(state); err != nil {
			return err
		}
	}
	nbrLines := state.CutBuffer.Len()
	if nbrLines > 0 {
		appendLines(startLineNbr, state, state.CutBuffer)
		state.changedSinceLastWrite = true
		state.addUndo(startLineNbr+1, startLineNbr+nbrLines, commandDelete, nil, cmd)
	}
	return nil
}

/*
CmdRead reads file and appends it after the addressed line.

 If file is not specified, then the default filename is used.
 If there is no default filename prior to the command, then the default filename is set to file.
 Otherwise, the default filename is unchanged.

 The address '0' (zero) is valid for this command; it reads the file at the beginning of the buffer.

 The current address is set to the address of the last line read or, if there were none, to the addressed line.
*/
func (cmd Command) CmdRead(state *State) error {
	filename, err := getFilename(strings.TrimSpace(cmd.restOfCmd), state, false)
	if err != nil {
		return err
	}
	var startLineNbr int
	// default is append at eof
	if !cmd.AddrRange.IsAddressRangeSpecified() {
		startLineNbr = state.Buffer.Len()
	} else {
		startLineNbr, err = cmd.AddrRange.start.calculateActualLineNumber(state)
		if err != nil {
			return err
		}
	}
	nbrBytesRead, listOfLines, err := ReadFile(filename)
	if err != nil {
		return err
	}
	fmt.Println(nbrBytesRead)
	nbrLinesRead := listOfLines.Len()
	if nbrLinesRead > 0 {
		appendLines(startLineNbr, state, listOfLines)
		state.changedSinceLastWrite = true
		state.addUndo(startLineNbr+1, startLineNbr+listOfLines.Len(), commandDelete, nil, cmd)
	} else {
		moveToLine(startLineNbr, state)
	}
	return nil
}

/*
CmdScroll scrolls n lines at a time starting at addressed line, and sets window size to n.
 The current address is set to the address of the last line printed.

 If n is not specified, then the current window size is used.

 Window size defaults to screen size minus two lines, or to 22 if screen size can't be determined.
*/
func (cmd Command) CmdScroll(state *State) error {
	var startLineNbr, endLineNbr int
	var err error
	if !cmd.AddrRange.IsAddressRangeSpecified() {
		startLineNbr = state.lineNbr + 1
	} else {
		startLineNbr, err = cmd.AddrRange.start.calculateActualLineNumber(state)
		if err != nil {
			return err
		}
	}
	// check for 'z<n>'
	if cmd.restOfCmd != "" {
		// parse to number if possible
		newWindowSize, err := strconv.Atoi(strings.TrimSpace(cmd.restOfCmd))
		if err != nil || newWindowSize < 1 {
			return errInvalidWindowSize
		}
		state.WindowSize = newWindowSize
	}
	endLineNbr = startLineNbr + state.WindowSize
	// sanitize
	if startLineNbr == 0 {
		startLineNbr = 1
	}
	if startLineNbr > state.Buffer.Len() {
		startLineNbr = state.Buffer.Len()
	}
	if endLineNbr > state.Buffer.Len() {
		endLineNbr = state.Buffer.Len()
	}
	startAddr := Address{startLineNbr, 0}
	endAddr := Address{endLineNbr, 0}
	cmd.AddrRange = AddressRange{startAddr, endAddr}
	return _printRange(os.Stdout, cmd, state, true)
}

/*
CmdTransfer copies (i.e., transfers) the addressed lines to after the right-hand destination address.

 If the destination address is 0, the lines are copied at the beginning of the buffer.

 The current address is set to the address of the last line copied.
*/
func (cmd Command) CmdTransfer(state *State) error {
	startLineNbr, endLineNbr, err := cmd.AddrRange.getAddressRange(state)
	if err != nil {
		return err
	}
	var destLineNbr int
	// default is current line for destination
	if destStr := strings.TrimSpace(cmd.restOfCmd); destStr == "" {
		destLineNbr = state.lineNbr
	} else {
		var destLine Address
		if destLine, err = newAddress(destStr); err != nil {
			return errInvalidDestinationAddress
		}
		if destLineNbr, err = destLine.calculateActualLineNumber(state); err != nil {
			return err
		}
	}
	if destLineNbr > state.Buffer.Len() {
		return errInvalidDestinationAddress
	}
	tempBuffer := copyLines(startLineNbr, endLineNbr, state)
	appendLines(destLineNbr, state, tempBuffer)
	state.changedSinceLastWrite = true

	// the undo is a delete command from destLineNbr + 1
	state.addUndo(destLineNbr+1, destLineNbr+tempBuffer.Len(), commandDelete, nil, cmd)
	return nil
}

/*
CmdUndo undoes the previous command.
*/
func (cmd Command) CmdUndo(state *State) error {

	if state.undo.Len() == 0 {
		return errNothingToUndo
	}

	undoEl := state.undo.Front()
	state.undo.Remove(undoEl)
	undo := undoEl.Value.(Undo)

	if state.Debug {
		fmt.Println(undo.cmd)
	}
	// set global flag to indicate we're undoing
	state.processingUndo = true
	var err error
	// cater for the 'special' undo commands
	switch undo.cmd.Cmd {
	case internalCommandUndoMove:
		err = handleUndoMove(undo, state)
	case internalCommandUndoSubst:
		err = handleUndoSubst(undo, state)
	default:
		_, err = ProcessCommand(undo.cmd, state, undo.text, false)
	}
	state.processingUndo = false
	return err
}

/*
CmdWrite handles the commands "w", "wq", and "W".

 Writes (or appends in case of W) the addressed lines to file.
 Any previous contents of file is lost without warning.

 If there is no default filename, then the default filename is set to file, otherwise it is unchanged.
 If no filename is specified, then the default filename is used.

 The current address is unchanged.

 In case of 'wq': a quit is performed immediately afterwards. (This is handled by the caller.)
*/
func (cmd Command) CmdWrite(state *State) error {
	// save current address
	currentLine := state.lineNbr

	// handle command sequence 'wq'
	filename := strings.TrimPrefix(cmd.restOfCmd, CommandQuit)
	filename, err := getFilename(strings.TrimSpace(filename), state, true)
	if err != nil {
		return err
	}

	var startLineNbr, endLineNbr int
	if !cmd.AddrRange.IsAddressRangeSpecified() {
		startLineNbr = 1
		endLineNbr = state.Buffer.Len()
	} else {
		startLineNbr, endLineNbr, err = cmd.AddrRange.calculateStartAndEndLineNumbers(state)
		if err != nil {
			return err
		}
	}
	// disallow 0
	if startLineNbr == 0 {
		return errInvalidLine
	}
	moveToLine(startLineNbr, state)
	nbrBytesWritten, err := WriteFile(filename, state.dotline, startLineNbr, endLineNbr)
	if err != nil {
		return err
	}
	fmt.Printf("%dC\n", nbrBytesWritten)
	state.changedSinceLastWrite = false
	moveToLine(currentLine, state)
	return nil
}

/*
CmdYank copies (yanks) the addressed lines to the cut buffer.

 The cut buffer is overwritten by subsequent 'c', 'd', 'j', 's', or 'y' commands.
 The current address is unchanged.
*/
func (cmd Command) CmdYank(state *State) error {
	currentAddress := state.lineNbr // save for later
	startLineNbr, endLineNbr, err := cmd.AddrRange.getAddressRange(state)
	if err != nil {
		return err
	}
	if startLineNbr == 0 {
		return errInvalidLine
	}

	state.CutBuffer = copyLines(startLineNbr, endLineNbr, state)
	moveToLine(currentAddress, state)
	return nil
}

// ----------------------------------------------------------------------------
//
// helper functions
//
// ----------------------------------------------------------------------------

/*
 Implements the undo for the command 'move'.

 This consists of two operations, unlike all the others
  - first delete the moved lines
  - then re-insert
*/
func handleUndoMove(undoCmd Undo, state *State) error {
	// first the delete
	_ = deleteLines(undoCmd.cmd.AddrRange.start.addr, undoCmd.cmd.AddrRange.end.addr, state)
	// then append. The line to append at is stored in the original command
	appendLines(undoCmd.originalCmd.AddrRange.start.addr-1, state, undoCmd.text)

	return nil
}

/*
 Implements the undo for the command 'subst'.
 This is a list of 1..n undo commands (each of which is a 'change' command).
*/
func handleUndoSubst(toplevelUndoCmd Undo, state *State) error {
	// undo.text == a list of 'change' undo-commands, NOT a list of changed lines
	for el := toplevelUndoCmd.text.Front(); el != nil; el = el.Next() {
		undoCmd := el.Value.(Undo)
		if undoCmd.cmd.Cmd != commandChange {
			panic(fmt.Sprintf("expected 'change' command, got '%s'\n", undoCmd.cmd.Cmd))
		}
		undoCmd.cmd.CmdChange(state, undoCmd.text)
	}
	return nil
}

/*
 Appends the lines in the list 'newLines' to the current buffer, after line #lineNbr.

 Afterwards, the state's current line will be set to the last of the new lines just appended.

 Special cases:
   - appending at the last line
	- appending at "line 0" i.e. before the first line
*/
func appendLines(lineNbr int, state *State, newLines *list.List) {
	// return if newLines is empty
	if newLines.Len() == 0 {
		return
	}
	if lineNbr == state.Buffer.Len() {
		// append at end
		state.Buffer.PushBackList(newLines)
		moveToLine(state.Buffer.Len(), state)
	} else if lineNbr == 0 {
		// append at start
		state.Buffer.PushFrontList(newLines)
		moveToLine(newLines.Len(), state)
	} else {
		moveToLine(lineNbr, state)
		nbrLinesEntered := 0
		mark := state.dotline
		for e := newLines.Front(); e != nil; e = e.Next() {
			mark = state.Buffer.InsertAfter(e.Value, mark)
			nbrLinesEntered++
		}
		moveToLine(lineNbr+nbrLinesEntered, state)
	}
}

/*
 Copies the required lines into a new list.
*/
func copyLines(startLineNbr, endLineNbr int, state *State) *list.List {
	tempBuffer := list.New()
	copyFunc := func(lineNbr int, el *list.Element, state *State) {
		tempBuffer.PushBack(el.Value)
	}
	iterateLines(startLineNbr, endLineNbr, state, copyFunc)
	return tempBuffer
}

/*
 Deletes the required lines from the state.buffer and returns them as a new list.
*/
func deleteLines(startLineNbr, endLineNbr int, state *State) (newList *list.List) {
	tempBuffer := list.New()
	deleteFunc := func(lineNbr int, el *list.Element, state *State) {
		state.Buffer.Remove(el)
		tempBuffer.PushBack(el.Value)
	}
	iterateLines(startLineNbr, endLineNbr, state, deleteFunc)
	return tempBuffer
}

/*
A LineProcessorFn is a function which operates on a line
*/
type LineProcessorFn func(lineNbr int, el *list.Element, state *State)

/*
 Iterate over the required lines and apply the given function.
*/
func iterateLines(startLineNbr, endLineNbr int, state *State, fn LineProcessorFn) {
	moveToLine(startLineNbr, state)
	el := state.dotline
	for lineNbr := startLineNbr; lineNbr <= endLineNbr; lineNbr++ {
		elementCopy := el
		el = el.Next()
		fn(lineNbr, elementCopy, state)
	}
}

func readInputLines() (newLines *list.List, nbrLinesEntered int, err error) {
	newLines = list.New()
	reader := bufio.NewReader(os.Stdin)
	nbrLinesEntered = 0
	for quit := false; !quit; {
		var inputStr string
		inputStr, err = reader.ReadString('\n')
		if err != nil {
			return
		}
		if inputStr == ".\n" {
			quit = true
		} else {
			nbrLinesEntered++
			newLines.PushBack(Line{inputStr})
		}
	}
	return
}

/*
 If potentialFilename is set, returns this. If setDefault is TRUE, the state.defaultFilename will
 be set to this filename.

 Otherwise, returns the state.defaultFilename.

 It is an error if both potentialFilename and state.defaultFilename are empty.
*/
func getFilename(potentialFilename string, state *State, setDefault bool) (filename string, err error) {
	if potentialFilename == "" {
		// use default if set
		if state.defaultFilename != "" {
			filename = state.defaultFilename
		} else {
			return "", errMissingFilename
		}
	} else {
		filename = potentialFilename
		if setDefault {
			state.defaultFilename = potentialFilename
		}
	}
	return filename, nil
}

func _printRange(writer io.Writer, cmd Command, state *State, printLineNumbers bool) error {
	startLineNbr, endLineNbr, err := cmd.AddrRange.calculateStartAndEndLineNumbers(state)
	if err != nil {
		return err
	}
	// disallow 0p
	if startLineNbr == 0 {
		return errInvalidLine
	}
	if endLineNbr == 0 {
		endLineNbr = 1
	}

	if startLineNbr > endLineNbr {
		panic(fmt.Sprintf("start line: %d, end line %d", startLineNbr, endLineNbr))
	}
	moveToLine(startLineNbr, state)

	el := state.dotline
	prevEl := el
	for lineNbr := startLineNbr; lineNbr <= endLineNbr; lineNbr++ {
		_printLine(writer, lineNbr, el.Value.(Line).Line, printLineNumbers)
		prevEl = el // store el, to be able to set dotline i/c we hit the end of the list
		el = el.Next()
	}
	// set dotline
	if el != nil {
		state.dotline = el
	} else {
		state.dotline = prevEl
	}
	state.lineNbr = endLineNbr
	return nil
}

func _printLine(writer io.Writer, lineNbr int, str string, printLineNumbers bool) {
	if printLineNumbers {
		fmt.Fprintf(writer, "%4d%c %s", lineNbr, '\t', str)
	} else {
		fmt.Fprintf(writer, str)
	}
}

/**
 * moves to the given line number and updates the state (dotline, lineNbr).
 */
func moveToLine(requiredLine int, state *State) {
	// TODO? always starts at the top of the file ...
	lineNbr := 1
	e := state.Buffer.Front()
	for ; e != nil; e = e.Next() {
		if requiredLine == lineNbr {
			break
		} else {
			lineNbr++
		}
	}
	// double check
	if requiredLine != lineNbr {
		panic(fmt.Sprintf("bad line number: got %d, wanted %d", lineNbr, requiredLine))
	}
	state.dotline = e
	state.lineNbr = lineNbr
	return
}

/*
 ProcessCommand processes the given command.

 enteredText is non-nil if we're processing an undo (e.g. undoing a delete)
 inGlobalCommand is set TRUE if we're already processing a 'g' command,
    in which case certain other commands are not allowed/do not make sense.

 Returns TRUE if the quit command has been given.
*/
func ProcessCommand(cmd Command, state *State, enteredText *list.List, inGlobalCommand bool) (quit bool, err error) {
	// following commands are not allowed whilst procesing a global "g" command
	if inGlobalCommand {
		switch cmd.Cmd {
		case CommandEdit, CommandEditUnconditionally,
			commandGlobal, commandGlobalInteractive,
			commandInverseGlobal, commandInverseGlobalInteractive,
			CommandHelp,
			CommandQuit, CommandQuitUnconditionally,
			CommandUndo, commandWrite, commandWriteAppend:
			return false, errNotAllowedInGlobalCommand
		default:
			//ok
		}
	}
	switch cmd.Cmd {
	case commandAppend, commandInsert:
		err = cmd.CmdAppendInsert(state, enteredText)
	case commandChange:
		err = cmd.CmdChange(state, enteredText)
	case commandDelete:
		err = cmd.CmdDelete(state, true)
	case CommandEdit:
		if state.changedSinceLastWrite {
			fmt.Println(unsavedChanges)
		} else {
			err = cmd.CmdEdit(state)
		}
	case CommandEditUnconditionally:
		err = cmd.CmdEdit(state)
	case CommandFilename:
		state.defaultFilename = strings.TrimSpace(cmd.restOfCmd)
	case commandGlobal:
		err = cmd.CmdGlobal(state)
	case commandGlobalInteractive:
		fmt.Println("not yet implemented")
	case CommandHelp:
		err = cmd.CmdHelp(state)
	case commandInverseGlobal:
		fmt.Println("not yet implemented")
	case commandInverseGlobalInteractive:
		fmt.Println("not yet implemented")
	case commandJoin:
		err = cmd.CmdJoin(state)
	case commandMark:
		fmt.Println("not yet implemented")
	case commandList:
		fmt.Println("not yet implemented")
	case commandMove:
		err = cmd.CmdMove(state)
	case commandNumber, commandPrint:
		err = cmd.CmdPrint(state)
	case CommandPrompt:
		state.ShowPrompt = !state.ShowPrompt
	case CommandQuit, CommandQuitUnconditionally:
		if cmd.Cmd == CommandQuit && state.changedSinceLastWrite {
			fmt.Println(unsavedChanges)
		} else {
			quit = true
		}
	case commandRead:
		err = cmd.CmdRead(state)
	case commandSubstitute:
		err = cmd.CmdSubstitute(state)
	case commandTransfer:
		err = cmd.CmdTransfer(state)
	case CommandUndo:
		err = cmd.CmdUndo(state)
	case commandWrite:
		err = cmd.CmdWrite(state)
		quit = (cmd.Cmd == commandWrite && strings.HasPrefix(cmd.restOfCmd, CommandQuit))
	case commandWriteAppend:
		fmt.Println("not yet implemented")
	case commandPut:
		err = cmd.CmdPut(state)
	case commandYank:
		err = cmd.CmdYank(state)
	case commandScroll:
		err = cmd.CmdScroll(state)
	case commandComment:
		err = cmd.CmdComment(state)
	case commandLinenumber:
		err = cmd.CmdLinenumber(state)
	case commandNoCommand:
		// nothing entered -- ignore
	default:
		fmt.Println("ERROR got command not in switch!?: ", cmd.Cmd)
	}
	return quit, err
}
