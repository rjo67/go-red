package main

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
const commandEdit string = "e"
const commandEditUnconditionally string = "E"
const commandFilename string = "f"
const commandGlobal string = "g"
const commandGlobalInteractive string = "G"
const commandInsert string = "i"
const commandJoin string = "j"
const commandMark string = "k"
const commandList string = "l" // print suffix
const commandMove string = "m"
const commandNumber string = "n" // print suffix
const commandPrint string = "p"  // print suffix
const commandPrompt string = "P"
const commandQuit string = "q"
const commandQuitUnconditionally string = "Q"
const commandRead string = "r"
const commandSubstitute string = "s"
const commandTransfer string = "t"
const commandUndo string = "u"
const commandInverseGlobal string = "v"
const commandInverseGlobalInteractive string = "V"
const commandWrite string = "w"
const commandWriteAppend string = "W"
const commandPut string = "x"
const commandYank string = "y"
const commandScroll string = "z"
const commandComment string = "#"
const commandLinenumber string = "="

// returned when an empty line was entered
const commandNoCommand string = ""

var unrecognisedCommand error = errors.New("unrecognised command")
var missingFilename error = errors.New("filename missing and no default set")
var invalidWindowSize error = errors.New("invalid window size")

var justNumberRE = regexp.MustCompile(`^\s*(\d*)\s*$`)
var commandLineRE = regexp.MustCompile("(.*?)([acdeEfgGijklmnpPqQrstuvVwWxyz#=])(.*)")

/*
 * A command, parsed from user input
 */
type Command struct {
	addrRange AddressRange
	cmd       string
	restOfCmd string
}

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
		addrString := matches[1]
		addrRange, err := newRange(addrString)
		if err != nil {
			return Command{}, err
		} else {
			return Command{addrRange, commandPrint, ""}, nil
		}
	}
	matches = commandLineRE.FindStringSubmatch(str)
	if matches != nil {
		addrString := matches[1]
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
		return Command{}, unrecognisedCommand
	}
}

/*
 * Appends text to the buffer after the addressed line.
 * or
 * Inserts text in the buffer before the addressed line.
 *
 * The address '0' (zero) is valid for this command;
 *   append: it places the entered text at the beginning of the buffer.
 *   insert: it is equivalent to address '1'.
 *
 * The current address is set to the address of the last line entered or,
 * if there were none, to the addressed line.
 */
func (cmd Command) CmdAppendInsert(state *State) error {
	// calc required line (checks if this is a valid line)
	lineNbr, err := calculateActualLineNumber(cmd.addrRange.start, state)
	if err != nil {
		return err
	}

	newLines, nbrLinesEntered, err := readInputLines()
	if nbrLinesEntered == 0 {
		return nil
	}
	state.changedSinceLastWrite = true

	// special case: append line 0
	// or insert with line 0 or 1

	if (cmd.cmd == commandAppend && lineNbr == 0) || (cmd.cmd == commandInsert && lineNbr <= 1) {
		state.buffer.PushFrontList(newLines)
		moveToLine(nbrLinesEntered, state)
	} else {
		// an "insert" at line <n> is the same as an append at line <n-1>
		if cmd.cmd == commandInsert {
			lineNbr--
		}
		appendLines(lineNbr, state, newLines)
	}

	return nil
}

/*
 Changes lines in the buffer.

 The addressed lines are deleted from the buffer, and text is inserted in their place.

 The current address is set to the address of the last line entered or, if there were none,
 to the new address of the line after the last line deleted;
 if the lines deleted were originally at the end of the buffer,
 the current address is set to the address of the new last line;
 if no lines remain in the buffer, the current address is set to zero.
*/
func (cmd Command) CmdChange(state *State) error {

	startLineNbr, err := calculateActualLineNumber(cmd.addrRange.start, state)
	if err != nil {
		return err
	}
	if startLineNbr == 0 {
		return invalidLine
	}

	// delete the lines
	deleteCmd := Command{cmd.addrRange, commandDelete, cmd.restOfCmd}
	deleteCmd.CmdDelete(state)

	var atEof bool

	// adjust the starting lineNbr if we've deleted at the end of the file
	if startLineNbr > state.buffer.Len() {
		startLineNbr = state.buffer.Len()
		atEof = true
	}

	newLines, nbrLinesEntered, err := readInputLines()
	if nbrLinesEntered == 0 {
		return nil
	}
	state.changedSinceLastWrite = true

	if atEof {
		appendLines(startLineNbr, state, newLines)
	} else {
		appendLines(startLineNbr-1, state, newLines)
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
	if lineNbr == state.buffer.Len() {
		// append at end
		state.buffer.PushBackList(newLines)
		moveToLine(state.buffer.Len(), state)
	} else if lineNbr == 0 {
		// append at start
		state.buffer.PushFrontList(newLines)
		moveToLine(newLines.Len(), state)
	} else {
		moveToLine(lineNbr, state)
		nbrLinesEntered := 0
		mark := state.dotline
		for e := newLines.Front(); e != nil; e = e.Next() {
			mark = state.buffer.InsertAfter(e.Value, mark)
			nbrLinesEntered++
		}
		moveToLine(lineNbr+nbrLinesEntered, state)
	}
}

/*
 Copies (yanks) the addressed lines to the cut buffer.

 The cut buffer is overwritten by subsequent 'c', 'd', 'j', 's', or 'y' commands.
 The current address is unchanged.
*/
func (cmd Command) CmdYank(state *State) error {
	currentAddress := state.lineNbr // save for later
	var startLineNbr, endLineNbr int
	var err error
	if !cmd.addrRange.isAddressRangeSpecified() {
		startLineNbr = state.lineNbr
		endLineNbr = state.lineNbr
	} else {
		startLineNbr, endLineNbr, err = calculateStartAndEndLineNumbers(cmd.addrRange, state)
	}
	if err != nil {
		return err
	}
	if startLineNbr == 0 {
		return invalidLine
	}

	state.cutBuffer = copyLines(startLineNbr, endLineNbr, state)
	moveToLine(currentAddress, state)
	return nil
}

/*
 Copies the required lines into a new list.
*/
func copyLines(startLineNbr, endLineNbr int, state *State) *list.List {
	moveToLine(startLineNbr, state)
	tempBuffer := list.New()
	el := state.dotline
	for lineNbr := startLineNbr; lineNbr <= endLineNbr; lineNbr++ {
		element := el
		el = el.Next()
		tempBuffer.PushBack(element.Value)
	}
	return tempBuffer
}

/*
 Deletes the required lines from the state.buffer and returns them as a new list.
*/
func deleteLines(startLineNbr, endLineNbr int, state *State) (newList *list.List) {
	moveToLine(startLineNbr, state)
	tempBuffer := list.New()
	el := state.dotline
	for lineNbr := startLineNbr; lineNbr <= endLineNbr; lineNbr++ {
		elementToDelete := el
		el = el.Next()
		state.buffer.Remove(elementToDelete)
		tempBuffer.PushBack(elementToDelete.Value)
	}
	return tempBuffer
}

/*
 Deletes the addressed lines from the buffer.

 The current address is set to the new address of the line after the last line deleted;
 if the lines deleted were originally at the end of the buffer,
   the current address is set to the address of the new last line;
 if no lines remain in the buffer, the current address is set to zero.

 Deleted lines are stored in the state.cutBuffer.
*/
func (cmd Command) CmdDelete(state *State) error {
	startLineNbr, endLineNbr, err := calculateStartAndEndLineNumbers(cmd.addrRange, state)
	if err != nil {
		return err
	}
	if startLineNbr == 0 {
		return invalidLine
	}

	tempBuffer := deleteLines(startLineNbr, endLineNbr, state)

	if tempBuffer.Len() == 0 {
		return nil
	}

	state.cutBuffer = tempBuffer
	state.changedSinceLastWrite = true

	newLineNbr := startLineNbr
	bufferLen := state.buffer.Len()
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
 Prints the line number of the addressed line.

 The current address is unchanged.
*/
func (cmd Command) CmdLinenumber(state *State) error {
	startLineNbr, err := calculateActualLineNumber(cmd.addrRange.start, state)
	if err != nil {
		return err
	}
	fmt.Println(startLineNbr)
	return nil
}

/*
  Edits file, and sets the default filename.
  If file is not specified, then the default filename is used.
  Any lines in the buffer are deleted before the new file is read.
  The current address is set to the address of the last line in the buffer.
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
	state.buffer = listOfLines
	state.changedSinceLastWrite = false
	moveToLine(state.buffer.Len(), state)
	return nil
}

/*
 Reads file and appends it after the addressed line.

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
	if !cmd.addrRange.isAddressRangeSpecified() {
		startLineNbr = state.buffer.Len()
	} else {
		startLineNbr, err = calculateActualLineNumber(cmd.addrRange.start, state)
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
	} else {
		moveToLine(startLineNbr, state)
	}
	return nil
}

/*
 Joins the addressed lines, replacing them by a single line containing their joined text.

 If only one address is given, this command does nothing.

 If lines are joined, the current address is set to the address of the joined line.
 Else, the current address is unchanged.
*/
func (cmd Command) CmdJoin(state *State) error {
	var startLineNbr, endLineNbr int
	var err error

	if !cmd.addrRange.isAddressRangeSpecified() {
		return nil
	} else {
		if startLineNbr, endLineNbr, err = calculateStartAndEndLineNumbers(cmd.addrRange, state); err != nil {
			return err
		}
	}
	moveToLine(startLineNbr, state)
	var sb strings.Builder
	el := state.dotline
	for lineNbr := startLineNbr; lineNbr <= endLineNbr; lineNbr++ {
		line := el.Value.(Line).line
		// replace the newline with a space
		sb.WriteString(strings.Replace(line, "\n", " ", 1))
		el = el.Next()
	}
	// add newline again
	sb.WriteString("\n")

	// delete the lines (and set cutbuffer)
	deleteCmd := Command{cmd.addrRange, commandDelete, cmd.restOfCmd}
	deleteCmd.CmdDelete(state)

	// append the string buffer at line <startLineNbr - 1> with the string buffer
	newLines := list.New()
	newLines.PushBack(Line{sb.String()})

	appendLines(startLineNbr-1, state, newLines)
	return nil
}

/*
 Moves lines in the buffer.

 The addressed lines are moved to after the right-hand destination address.
 The destination address '0' (zero) is valid for this command;
    it moves the addressed lines to the beginning of the buffer.

 It is an error if the destination address falls within the range of moved lines.

 The current address is set to the new address of the last line moved.
*/
func (cmd Command) CmdMove(state *State) error {
	var startLineNbr, endLineNbr int
	var err error
	// default is current line (for both start/end, and dest)
	if !cmd.addrRange.isAddressRangeSpecified() {
		startLineNbr = state.lineNbr
		endLineNbr = state.lineNbr
	} else {
		if startLineNbr, endLineNbr, err = calculateStartAndEndLineNumbers(cmd.addrRange, state); err != nil {
			return err
		}
	}
	var destLineNbr int
	// default is current line for destination
	if destStr := strings.TrimSpace(cmd.restOfCmd); destStr == "" {
		destLineNbr = state.lineNbr
	} else {
		var destLine Address
		if destLine, err = newAddress(destStr); err != nil {
			return invalidDestinationAddress
		}
		if destLineNbr, err = calculateActualLineNumber(destLine, state); err != nil {
			return err
		}
	}
	if startLineNbr == 0 {
		return invalidDestinationAddress
	}
	if destLineNbr > state.buffer.Len() {
		return invalidDestinationAddress
	}
	// it is an error if the destination address falls within the range of moved lines
	if destLineNbr >= startLineNbr && destLineNbr < endLineNbr {
		return invalidDestinationAddress
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
	state.changedSinceLastWrite = true
	return nil
}

/*
 Copies (i.e., transfers) the addressed lines to after the right-hand destination address.

 If the destination address is 0, the lines are copied at the beginning of the buffer.

 The current address is set to the address of the last line copied.
*/
func (cmd Command) CmdTransfer(state *State) error {
	var startLineNbr, endLineNbr int
	var err error
	// default is current line (for start and end)
	if !cmd.addrRange.isAddressRangeSpecified() {
		startLineNbr = state.lineNbr
		endLineNbr = state.lineNbr
	} else {
		if startLineNbr, endLineNbr, err = calculateStartAndEndLineNumbers(cmd.addrRange, state); err != nil {
			return err
		}
	}
	var destLineNbr int
	// default is current line for destination
	if destStr := strings.TrimSpace(cmd.restOfCmd); destStr == "" {
		destLineNbr = state.lineNbr
	} else {
		var destLine Address
		if destLine, err = newAddress(destStr); err != nil {
			return invalidDestinationAddress
		}
		if destLineNbr, err = calculateActualLineNumber(destLine, state); err != nil {
			return err
		}
	}
	if destLineNbr > state.buffer.Len() {
		return invalidDestinationAddress
	}
	tempBuffer := copyLines(startLineNbr, endLineNbr, state)
	appendLines(destLineNbr, state, tempBuffer)
	state.changedSinceLastWrite = true
	return nil
}

/*
 Copies (puts) the contents of the cut buffer to after the addressed line.
 The current address is set to the address of the last line copied.
*/
func (cmd Command) CmdPut(state *State) error {
	var startLineNbr int
	var err error
	// default is append at current line
	if !cmd.addrRange.isAddressRangeSpecified() {
		startLineNbr = state.lineNbr
	} else {
		if startLineNbr, err = calculateActualLineNumber(cmd.addrRange.start, state); err != nil {
			return err
		}
	}
	nbrLines := state.cutBuffer.Len()
	if nbrLines > 0 {
		appendLines(startLineNbr, state, state.cutBuffer)
	}
	return nil
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
			return "", missingFilename
		}
	} else {
		filename = potentialFilename
		if setDefault {
			state.defaultFilename = potentialFilename
		}
	}
	return filename, nil
}

/*
 Handles the commands "w", "wq", and "W".

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
	filename := strings.TrimPrefix(cmd.restOfCmd, commandQuit)
	filename, err := getFilename(strings.TrimSpace(filename), state, true)
	if err != nil {
		return err
	}

	var startLineNbr, endLineNbr int
	if !cmd.addrRange.isAddressRangeSpecified() {
		startLineNbr = 1
		endLineNbr = state.buffer.Len()
	} else {
		startLineNbr, endLineNbr, err = calculateStartAndEndLineNumbers(cmd.addrRange, state)
		if err != nil {
			return err
		}
	}
	// disallow 0
	if startLineNbr == 0 {
		return invalidLine
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
 * Prints the addressed lines. The current address is set to the address of the last line printed.
 */
func (cmd Command) CmdPrint(state *State) error {
	return _printRange(os.Stdout, cmd, state, false)
}

/*
 Scrolls n lines at a time starting at addressed line, and sets window size to n.
 The current address is set to the address of the last line printed.

 If n is not specified, then the current window size is used.

 Window size defaults to screen size minus two lines, or to 22 if screen size can't be determined.
*/
func (cmd Command) CmdScroll(state *State) error {
	var startLineNbr, endLineNbr int
	var err error
	if !cmd.addrRange.isAddressRangeSpecified() {
		startLineNbr = state.lineNbr + 1
	} else {
		startLineNbr, err = calculateActualLineNumber(cmd.addrRange.start, state)
		if err != nil {
			return err
		}
	}
	// check for 'z<n>'
	if cmd.restOfCmd != "" {
		// parse to number if possible
		newWindowSize, err := strconv.Atoi(strings.TrimSpace(cmd.restOfCmd))
		if err != nil || newWindowSize < 1 {
			return invalidWindowSize
		}
		state.windowSize = newWindowSize
	}
	endLineNbr = startLineNbr + state.windowSize
	// sanitize
	if startLineNbr == 0 {
		startLineNbr = 1
	}
	if startLineNbr > state.buffer.Len() {
		startLineNbr = state.buffer.Len()
	}
	if endLineNbr > state.buffer.Len() {
		endLineNbr = state.buffer.Len()
	}
	startAddr := Address{startLineNbr, 0}
	endAddr := Address{endLineNbr, 0}
	cmd.addrRange = AddressRange{startAddr, endAddr}
	return _printRange(os.Stdout, cmd, state, true)
}

/*
 * Prints the addressed lines, preceding each line by its line number and a <tab>.
 * The current address is set to the address of the last line printed.
 */
func (cmd Command) CmdNumber(state *State) error {
	return _printRange(os.Stdout, cmd, state, true)
}

func _printRange(writer io.Writer, cmd Command, state *State, printLineNumbers bool) error {
	startLineNbr, endLineNbr, err := calculateStartAndEndLineNumbers(cmd.addrRange, state)
	if err != nil {
		return err
	}
	// disallow 0p
	if startLineNbr == 0 {
		return invalidLine
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
		_printLine(writer, lineNbr, el.Value.(Line).line, printLineNumbers)
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
	e := state.buffer.Front()
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
