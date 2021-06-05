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
const (
	commandAppend                   string = "a"
	commandChange                   string = "c"
	commandDelete                   string = "d"
	commandEdit                     string = "e"
	commandEditUnconditionally      string = "E"
	commandFilename                 string = "f"
	commandGlobal                   string = "g"
	commandGlobalInteractive        string = "G"
	commandHelp                     string = "h" // a startling departure from the ed range of commands ...
	commandInsert                   string = "i"
	commandJoin                     string = "j"
	commandMark                     string = "k"
	commandList                     string = "l" // print suffix
	commandMove                     string = "m"
	commandNumber                   string = "n" // print suffix
	commandPrint                    string = "p" // print suffix
	commandPrompt                   string = "P"
	commandQuit                     string = "q"
	commandQuitUnconditionally      string = "Q"
	commandRead                     string = "r"
	commandSubstitute               string = "s"
	commandTransfer                 string = "t"
	commandUndo                     string = "u"
	commandInverseGlobal            string = "v"
	commandInverseGlobalInteractive string = "V"
	commandWrite                    string = "w"
	commandWriteAppend              string = "W"
	commandPut                      string = "x"
	commandYank                     string = "y"
	commandScroll                   string = "z"
	commandComment                  string = "#"
	commandLinenumber               string = "="

	internalCommandUndoMove  string = ")" // an internal command to undo the 'move' command (which requires two steps)
	internalCommandUndoSubst string = "(" // an internal command to undo the 'subst' command (which is 1..n 'change' commands)
	commandNoCommand         string = ""  // returned when an empty line was entered
)

const unsavedChanges string = "buffer has unsaved changes"

var (
	errInvalidWindowSize         error = errors.New("invalid window size")
	errBadMarkname               error = errors.New("a name of a mark must be one char: a-z")
	errMissingFilename           error = errors.New("filename missing and no default set")
	errNotAllowedInGlobalCommand error = errors.New("command cannot be used within 'g'/'v'")
	errNothingToUndo             error = errors.New("nothing to undo")
	errUnrecognisedCommand       error = errors.New("unrecognised command")
	errAddressHasNotBeenResolved error = errors.New("address has not been resolved")
)

const (
	_simplifiedAddressRE = `([+-]?\d+|[\.\$\+-]|'[a-z]|\/[^\/]*\/|\?[^\?]*\?|\s*)+`
	_commandRE           = `[acdeEfgGhijklmnpPqQrstuvVwWxyz#=]`
)

var (
	singleLetterRE = regexp.MustCompile(`^([a-z])$`)
	// This RE matches user input of the form addr1 sep addr2 cmd (everything is optional, whitespace allowed anywhere)
	// The group 'addrRange' will contain addr1 sep addr2.
	// The group 'cmd' will contain everything else (note: a command is optional)
	// In case of syntax errors (e.g. nonterminated regex, mark followed by number), the 'cmd' group will contain the string starting at the error
	commandLineRE = regexp.MustCompile(
		"^(?P<addrRange>" + _simplifiedAddressRE +
			"[,;]?" + _simplifiedAddressRE +
			")(?P<cmd>" + _commandRE + "?)(?P<rest>.*)$")
)

type resolvedAddress struct {
	start, end int
}

/*
Command stores the command which has been parsed from user input.
*/
type Command struct {
	addrRange         AddressRange    // address range as parsed from user input
	parsedAddrString  string          // the string entered for the address range  (for debugging purposes)
	addressIsResolved bool            // will be 'false' by default; true implies resolvedAddress has been set
	resolved          resolvedAddress // resolved addresses
	cmd               string          // command identifier
	restOfCmd         string          // rest of command, if present
}

func (cmd *Command) resolveAddress(state *State) error {
	start, end, err := cmd.addrRange.calculateStartAndEndLineNumbers(state.lineNbr, state.Buffer)
	if err != nil {
		return err
	}
	cmd.resolved = resolvedAddress{start: start, end: end}
	cmd.addressIsResolved = true
	return nil
}

/*
Copies the current command into a new object, using the resolved addresses,
but setting the rest of the fields according to the parameters.
*/
func (cmd *Command) createNewResolvedCommand(newCmdIdent, newRestOfCmd string) (Command, error) {
	var newCmd Command
	if !cmd.addressIsResolved {
		return newCmd, errAddressHasNotBeenResolved
	}
	newCmd = Command{addressIsResolved: true, resolved: resolvedAddress{start: cmd.resolved.start, end: cmd.resolved.end},
		cmd: newCmdIdent, restOfCmd: newRestOfCmd}
	return newCmd, nil
}

/*
ParseCommand parses the given string and creates a Command object.
*/
func ParseCommand(str string, debug bool) (cmd Command, err error) {
	if debug {
		fmt.Printf("ParseCommand, str: '%s'\n", str)
	}
	if strings.TrimSpace(str) == "" {
		// newline alone == +1p
		str = "+1p"
	}
	matches := findNamedMatches(commandLineRE, str, true)
	/*
		if matches == nil {
			// add on implicit "p" and try again
			matches = commandLineRE.FindStringSubmatch(str + "p")
		}
	*/
	// the RE really always matches. In case of a syntax error, 'addrRange' and 'cmd' will be empty and 'rest' will be filled
	if matches != nil {
		addrString := strings.TrimSpace(matches["addrRange"])
		cmdString := strings.TrimSpace(matches["cmd"])
		restOfCmd := strings.TrimSpace(matches["rest"])
		if debug {
			fmt.Printf("parsed addrString: '%s', cmd: '%s', rest: %s\n", addrString, cmdString, restOfCmd)
		}

		if len(cmdString) == 0 && len(restOfCmd) != 0 && (restOfCmd[0:1] == "/" || restOfCmd[0:1] == "?") {
			return Command{}, fmt.Errorf("could not parse command (non-terminated regex?)")
		}
		addrRange, err := newRange(addrString)
		if err != nil {
			return Command{}, fmt.Errorf("could not parse address range: %w", err)
		} else {
			// if cmdString is empty, then use implicit "p"
			if len(cmdString) == 0 {
				// in this case, make sure restOfCmd is also empty
				if len(restOfCmd) != 0 {
					return Command{}, fmt.Errorf("general parse error")
				}
				cmdString = "p"
			} else {
				// check command
				var cmdMatch bool
				if cmdMatch, err = regexp.MatchString(_commandRE, cmdString); err != nil {
					return Command{}, fmt.Errorf("RE error parsing command: '%s', error: %w", cmdString, err)
				}
				if !cmdMatch {
					return Command{}, fmt.Errorf("could not parse command: '%s'", cmdString)
				}
			}
			cmd := Command{parsedAddrString: addrString, addrRange: addrRange, cmd: cmdString, restOfCmd: restOfCmd}
			if debug {
				fmt.Printf("parsed cmd: '%v'\n", cmd)
			}
			return cmd, nil
		}
	} else {
		return Command{}, errUnrecognisedCommand
	}
}

/*
AppendInsert appends text to the buffer after the addressed line.
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
func (cmd Command) AppendInsert(state *State, inputLines *list.List) error {
	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}

	var newLines *list.List
	var nbrLinesEntered int
	var err error
	if inputLines != nil {
		newLines = inputLines
		nbrLinesEntered = inputLines.Len()
	} else {
		if newLines, nbrLinesEntered, err = readInputLines(); err != nil {
			return err
		}
	}
	if nbrLinesEntered == 0 {
		return nil
	}

	state.changedSinceLastWrite = true

	// special case: append line 0
	// or insert with line 0 or 1

	if (cmd.cmd == commandAppend && cmd.resolved.start == 0) || (cmd.cmd == commandInsert && cmd.resolved.start <= 1) {
		state.Buffer.PushFrontList(newLines)
		moveToLine(nbrLinesEntered, state)
		if !state.processingUndo {
			state.addUndo(1, nbrLinesEntered, commandDelete, newLines, cmd)
		}
	} else {
		var startAddrForUndo, endAddrForUndo int
		lineNbr := cmd.resolved.start
		// an "insert" at line <n> is the same as an append at line <n-1>
		if cmd.cmd == commandInsert {
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
Change changes lines in the buffer.

 The addressed lines are deleted from the buffer, and text is inserted in their place.

 The current address is set to the address of the last line entered or, if there were none,
 to the new address of the line after the last line deleted;
 if the lines deleted were originally at the end of the buffer,
 the current address is set to the address of the new last line;
 if no lines remain in the buffer, the current address is set to zero.
*/
func (cmd Command) Change(state *State, inputLines *list.List) error {
	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}
	if cmd.resolved.start == 0 {
		return fmt.Errorf("change: %w", errorInvalidLine("start line is 0", nil))
	}

	var (
		newLines        *list.List
		nbrLinesEntered int
		err             error
	)
	if inputLines != nil {
		newLines = inputLines
		nbrLinesEntered = inputLines.Len()
	} else {
		// get the input, abort if empty
		if newLines, nbrLinesEntered, err = readInputLines(); err != nil {
			return err
		}
	}
	if nbrLinesEntered == 0 {
		return nil
	}

	// delete the lines
	deleteCmd, err := cmd.createNewResolvedCommand(commandDelete, cmd.restOfCmd)
	if err != nil {
		return err
	}
	if err = deleteCmd.Delete(state, false); err != nil {
		return err
	}
	// what's deleted is stored in the cutBuffer

	var atEOF bool

	// adjust the starting lineNbr if we've deleted at the end of the file
	startLineNbr := cmd.resolved.start
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
Comment processes comments.
*/
func (cmd Command) Comment(state *State) error {
	// does nothing
	return nil
}

/*
Delete deletes the addressed lines from the buffer.

 The current address is set to the new address of the line after the last line deleted;
 if the lines deleted were originally at the end of the buffer,
   the current address is set to the address of the new last line;
 if no lines remain in the buffer, the current address is set to zero.

 Deleted lines are stored in the state.CutBuffer.

 If addUndo is true, an undo command will be stored in state.undo.
 (This will be affected by the value of state.processingUndo)
*/
func (cmd Command) Delete(state *State, addUndo bool) error {
	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}
	if cmd.resolved.start == 0 {
		return fmt.Errorf("delete: %w", errorInvalidLine("start line is 0", nil))
	}

	tempBuffer := deleteLines(cmd.resolved.start, cmd.resolved.end, state)

	if tempBuffer.Len() == 0 {
		return nil
	}

	state.CutBuffer = tempBuffer
	state.changedSinceLastWrite = true
	bufferLen := state.Buffer.Len()

	state.updateMarks(commandDelete, cmd.resolved.start, cmd.resolved.end, -1)

	// inverse of delete m..n  ist insert at m
	if addUndo {
		// special case: we've deleted the last line
		if cmd.resolved.start > bufferLen {
			// undo of $d is $-1,a
			state.addUndo(endOfFile, endOfFile, commandAppend, tempBuffer, cmd)
		} else {
			state.addUndo(cmd.resolved.start, cmd.resolved.start, commandInsert, tempBuffer, cmd)
		}
	}

	// set up line nbr and dotline

	newLineNbr := cmd.resolved.start
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
Edit reads in a file, and sets the default filename.
  If file is not specified, then the default filename is used.
  Any lines in the buffer are deleted before the new file is read.
  The current address is set to the address of the last line in the buffer.
  Resets undo buffer.
*/
func (cmd Command) Edit(state *State) error {
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
Join joins the addressed lines, replacing them by a single line containing their joined text.

 If only one address is given, this command does nothing.

 If lines are joined, the current address is set to the address of the joined line.
 Else, the current address is unchanged.

 Calls internally Change, which is where the undo is handled.
*/
func (cmd Command) Join(state *State) error {
	var err error

	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}
	var sb strings.Builder
	joinFn := func(lineNbr int, el *list.Element, state *State) {
		line := el.Value.(Line).Line
		// replace the newline with a space
		sb.WriteString(strings.Replace(line, "\n", " ", 1))
	}
	iterateLines(cmd.resolved.start, cmd.resolved.end, state, joinFn)
	// the last line has had an extra space added; remove this and add newline again
	joinedLines := sb.String()[0:sb.Len()-1] + "\n"

	changeCommand, err := cmd.createNewResolvedCommand(commandChange, cmd.restOfCmd)
	if err != nil {
		return err
	}
	newLines := list.New()
	newLines.PushBack(Line{joinedLines})
	err = changeCommand.Change(state, newLines)
	return err
}

/*
Linenumber prints the line number of the addressed line.

 The current address is unchanged.
*/
func (cmd Command) Linenumber(state *State) error {
	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}
	fmt.Println(cmd.resolved.start)
	return nil
}

/*
Mark marks the given line.

Name of the mark must be one char (a..z)
 It is an error if an address range is specified.

 The current address is unchanged.
*/
func (cmd Command) Mark(state *State) error {
	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}
	matches := singleLetterRE.FindStringSubmatch(strings.TrimSpace(cmd.restOfCmd))
	if matches == nil {
		return errBadMarkname
	}
	markName := matches[1]
	if cmd.addrRange.end.isSpecified() {
		return ErrRangeMayNotBeSpecified
	}
	state.addMark(markName, cmd.resolved.start)
	return nil
}

/*
Move moves lines in the buffer.

 The addressed lines are moved to after the right-hand destination address (stored in cmd.restOfCmd).
 The destination address '0' (zero) is valid for this command;
    it moves the addressed lines to the beginning of the buffer.

 It is an error if the destination address falls within the range of moved lines.

 The current address is set to the new address of the last line moved.

 Undo is handled by storing the special command 'internalCommandUndoMove'.
*/
func (cmd Command) Move(state *State) error {
	// default is current line (for both start/end, and dest)
	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}
	var destLineNbr int
	var err error
	// default is current line for destination
	destStr := strings.TrimSpace(cmd.restOfCmd)
	if destStr == "" {
		destLineNbr = state.lineNbr
	} else {
		var destLine Address
		if destLine, err = newAddress(destStr); err != nil {
			return errorInvalidDestination(destStr, err)
		}
		if destLineNbr, err = destLine.calculateActualLineNumber(state.lineNbr, state.Buffer); err != nil {
			return errorInvalidDestination(destStr, err)
		}
	}
	startLineNbr := cmd.resolved.start
	if startLineNbr == 0 {
		return errorInvalidLine("start line=0", nil)
	}
	if destLineNbr > state.Buffer.Len() {
		return errorInvalidDestination(fmt.Sprintf("dest line: %d > last line: %d", destLineNbr, state.Buffer.Len()), nil)
	}
	// it is an error if the destination address falls within the range of moved lines
	if destLineNbr >= startLineNbr && destLineNbr < cmd.resolved.end {
		return errorInvalidDestination(fmt.Sprintf("dest line: %d is in range of moved lines: %d..%d", destLineNbr, startLineNbr, cmd.resolved.end), nil)
	}

	// delete the lines
	tempBuffer := deleteLines(startLineNbr, cmd.resolved.end, state)
	if tempBuffer.Len() == 0 {
		return nil
	}

	state.updateMarks(commandMove, startLineNbr, cmd.resolved.end, destLineNbr)

	// adjust destination line number if it has been affected by the delete
	if destLineNbr >= startLineNbr {
		destLineNbr -= (cmd.resolved.end - startLineNbr + 1)
	}

	appendLines(destLineNbr, state, tempBuffer)
	state.addUndo(destLineNbr+1, destLineNbr+tempBuffer.Len(), internalCommandUndoMove, tempBuffer, cmd)
	state.changedSinceLastWrite = true
	return nil
}

/*
Print prints the addressed lines.

 For command "n": Precedes each line by its line number and a <tab>.

 The current address is set to the address of the last line printed.
*/
func (cmd Command) Print(state *State) error {
	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}
	// no address specified defaults to .
	// TODO don't think this can occur anymore
	if !cmd.addrRange.IsSpecified() {
		cmd.addrRange = newValidRange(identDot)
	}
	return _printRange(os.Stdout, cmd.resolved.start, cmd.resolved.end, state, cmd.cmd == commandNumber)
}

/*
Put copies (puts) the contents of the cut buffer to after the addressed line.
If no address was specified, defaults to ".".
 The current address is set to the address of the last line copied.
*/
func (cmd Command) Put(state *State) error {
	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}
	if cmd.resolved.start == 0 {
		return fmt.Errorf("put: %w", errorInvalidLine("start line is 0", nil))
	}
	// range not allowed
	if cmd.resolved.start != cmd.resolved.end {
		return fmt.Errorf("put: %w", ErrRangeMayNotBeSpecified)
	}

	startLineNbr := cmd.resolved.start
	// default is append at current line, 'override' cmd.resolved.start if necessary
	if cmd.addrRange.start.isNotSpecified() {
		startLineNbr = state.lineNbr
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
Read reads file and appends it after the addressed line.

 If file is not specified, then the default filename is used.
 If there is no default filename prior to the command, then the default filename is set to file.
 Otherwise, the default filename is unchanged.

 The address '0' (zero) is valid for this command; it reads the file at the beginning of the buffer.

 The current address is set to the address of the last line read or, if there were none, to the addressed line.
*/
func (cmd Command) Read(state *State) error {
	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}
	// range not allowed
	if cmd.addrRange.end.isSpecified() {
		return fmt.Errorf("read: %w", ErrRangeMayNotBeSpecified)
	}

	filename, err := getFilename(strings.TrimSpace(cmd.restOfCmd), state, false)
	if err != nil {
		return err
	}
	var startLineNbr int
	// default is append at eof
	if cmd.addrRange.start.isNotSpecified() {
		startLineNbr = state.Buffer.Len()
	} else {
		startLineNbr = cmd.resolved.start
	}
	nbrBytesRead, listOfLines, err := ReadFile(filename)
	if err != nil {
		return err
	}
	fmt.Printf("%dL, %dC\n", listOfLines.Len(), nbrBytesRead)
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
Scroll scrolls n lines at a time starting at addressed line, and sets window size to n.
 The current address is set to the address of the last line printed.

 If n is not specified, then the current window size is used.

 Window size defaults to screen size minus two lines, or to 22 if screen size can't be determined.
*/
func (cmd Command) Scroll(state *State) error {
	return cmd._scroll(state, os.Stdout)
}
func (cmd Command) _scroll(state *State, writer io.Writer) error {
	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}
	if !cmd.addrRange.end.isNotSpecified() {
		return fmt.Errorf("scroll: cannot specify an address range")
	}
	var startLineNbr, endLineNbr int
	if cmd.addrRange.start.isNotSpecified() {
		startLineNbr = state.lineNbr + 1
	} else {
		startLineNbr = cmd.resolved.start
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
	startLineNbr = minIntOf(startLineNbr, state.Buffer.Len())
	endLineNbr = minIntOf(endLineNbr, state.Buffer.Len())

	return _printRange(writer, startLineNbr, endLineNbr, state, true)
}

/*
Transfer copies (i.e. transfers) the addressed lines to after the right-hand destination address.

 If the destination address is 0, the lines are copied at the beginning of the buffer.

 The current address is set to the address of the last line copied.
*/
func (cmd Command) Transfer(state *State) error {
	startLineNbr, endLineNbr, err := cmd.addrRange.getAddressRange(state.lineNbr, state.Buffer)
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
			return errorInvalidDestination(fmt.Sprintf("transfer: error parsing destination address: %s", destStr), err)
		}
		if destLineNbr, err = destLine.calculateActualLineNumber(state.lineNbr, state.Buffer); err != nil {
			return err
		}
	}
	if destLineNbr > state.Buffer.Len() {
		return errorInvalidDestination(fmt.Sprintf("transfer: destLine: %d > max line: %d", destLineNbr, state.Buffer.Len()), nil)
	}
	tempBuffer := copyLines(startLineNbr, endLineNbr, state)
	appendLines(destLineNbr, state, tempBuffer)
	state.changedSinceLastWrite = true

	// the undo is a delete command from destLineNbr + 1
	state.addUndo(destLineNbr+1, destLineNbr+tempBuffer.Len(), commandDelete, nil, cmd)
	return nil
}

/*
Undo undoes the previous command.
*/
func (cmd Command) Undo(state *State) error {

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
	switch undo.cmd.cmd {
	case internalCommandUndoMove:
		err = handleUndoMove(undo, state)
	case internalCommandUndoSubst:
		err = handleUndoSubst(undo, state)
	default:
		_, err = undo.cmd.ProcessCommand(state, undo.text, false)
	}
	state.processingUndo = false
	return err
}

/*
Write handles the commands "w", "wq", and "W".

 Writes (or appends in case of W) the addressed lines to file.
 Any previous contents of file is lost without warning.

 If there is no default filename, then the default filename is set to file, otherwise it is unchanged.
 If no filename is specified, then the default filename is used.

 The current address is unchanged.

 In case of 'wq': a quit is performed immediately afterwards. (This is handled by the caller.)
*/
func (cmd Command) Write(state *State) error {
	// save current address
	currentLine := state.lineNbr

	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}

	// handle command sequence 'wq'
	filename := strings.TrimPrefix(cmd.restOfCmd, commandQuit)
	filename, err := getFilename(strings.TrimSpace(filename), state, true)
	if err != nil {
		return err
	}

	var startLineNbr, endLineNbr int
	if !cmd.addrRange.IsSpecified() {
		startLineNbr = 1
		endLineNbr = state.Buffer.Len()
	} else {
		startLineNbr, endLineNbr = cmd.resolved.start, cmd.resolved.end
	}
	// disallow 0
	if startLineNbr == 0 {
		return fmt.Errorf("write: %w", errorInvalidLine("start line is 0", nil))
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
func (cmd Command) Yank(state *State) error {
	if !cmd.addressIsResolved {
		return errAddressHasNotBeenResolved
	}
	if cmd.resolved.start == 0 {
		return fmt.Errorf("yank: %w", errorInvalidLine("start line is 0", nil))
	}
	currentAddress := state.lineNbr // save for later

	state.CutBuffer = copyLines(cmd.resolved.start, cmd.resolved.end, state)
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
	// first the delete...
	undoStartLine, err := undoCmd.cmd.addrRange.start.calculateActualLineNumber(state.lineNbr, state.Buffer)
	if err != nil {
		return err
	}
	undoEndLine, err := undoCmd.cmd.addrRange.start.calculateActualLineNumber(state.lineNbr, state.Buffer)
	if err != nil {
		return err
	}
	_ = deleteLines(undoStartLine, undoEndLine, state)

	// ...then the append. The line to append at is stored in the original command
	originalStartLine, err := undoCmd.originalCmd.addrRange.start.calculateActualLineNumber(state.lineNbr, state.Buffer)
	if err != nil {
		return err
	}
	appendLines(originalStartLine-1, state, undoCmd.text)

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
		if undoCmd.cmd.cmd != commandChange {
			panic(fmt.Sprintf("expected 'change' command, got '%s'\n", undoCmd.cmd.cmd))
		}
		undoCmd.cmd.Change(state, undoCmd.text)
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

func _printRange(writer io.Writer, startLine, endLine int, state *State, printLineNumbers bool) error {
	// disallow 0p
	if startLine == 0 {
		return fmt.Errorf("print: %w", errorInvalidLine("start line is 0", nil))
	}
	if endLine == 0 {
		endLine = 1
	}
	if startLine > endLine {
		panic(fmt.Sprintf("start line: %d, end line %d", startLine, endLine))
	}
	moveToLine(startLine, state)

	el := state.dotline
	prevEl := el
	for lineNbr := startLine; lineNbr <= endLine; lineNbr++ {
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
	state.lineNbr = endLine
	return nil
}

func _printLine(writer io.Writer, lineNbr int, str string, printLineNumbers bool) {
	if printLineNumbers {
		fmt.Fprintf(writer, "%4d%c %s", lineNbr, '\t', str)
	} else {
		fmt.Fprint(writer, str)
	}
}

/**
 * Returns element in the buffer corresponding to the given line number.
 */
func _findLine(requiredLine int, buffer *list.List) *list.Element {
	// TODO? always starts at the top of the file ...
	lineNbr, e := 1, buffer.Front()
	for ; e != nil && lineNbr != requiredLine; e, lineNbr = e.Next(), lineNbr+1 {
	}
	// double check
	if requiredLine != lineNbr {
		panic(fmt.Sprintf("bad line number: got %d, wanted %d", lineNbr, requiredLine))
	}
	return e
}

/**
 * moves to the given line number and updates the state (dotline, lineNbr).
 */
func moveToLine(requiredLine int, state *State) {
	e := _findLine(requiredLine, state.Buffer)
	state.dotline = e
	state.lineNbr = requiredLine
}

/*
 ProcessCommand processes the given command.

 enteredText is non-nil if we're processing an undo (e.g. undoing a delete)
 inGlobalCommand is set TRUE if we're already processing a 'g' command,
    in which case certain other commands are not allowed/do not make sense.

 Returns TRUE if the quit command has been given.
*/
func (cmd Command) ProcessCommand(state *State, enteredText *list.List, inGlobalCommand bool) (quit bool, err error) {
	// following commands are not allowed whilst procesing a global "g" command
	if inGlobalCommand {
		switch cmd.cmd {
		case commandEdit, commandEditUnconditionally,
			commandGlobal, commandGlobalInteractive,
			commandInverseGlobal, commandInverseGlobalInteractive,
			commandHelp,
			commandQuit, commandQuitUnconditionally,
			commandUndo, commandWrite, commandWriteAppend:
			return false, errNotAllowedInGlobalCommand
		default:
			//ok
		}
	}
	// check for commands which cannot take ranges
	switch cmd.cmd {
	case commandEdit, commandEditUnconditionally,
		commandFilename, commandHelp, commandPrompt,
		commandQuit, commandQuitUnconditionally,
		commandUndo:
		if cmd.addrRange.IsSpecified() {
			err = ErrRangeMayNotBeSpecified
		}
	default:
		//ok
	}

	// first, resolve addresses
	if !cmd.addressIsResolved {
		if err = cmd.resolveAddress(state); err != nil {
			return false, err
		}
	}

	switch cmd.cmd {
	case commandAppend, commandInsert:
		err = cmd.AppendInsert(state, enteredText)
	case commandChange:
		err = cmd.Change(state, enteredText)
	case commandDelete:
		err = cmd.Delete(state, true)
	case commandEdit:
		if state.changedSinceLastWrite {
			fmt.Println(unsavedChanges)
		} else {
			err = cmd.Edit(state)
		}
	case commandEditUnconditionally:
		err = cmd.Edit(state)
	case commandFilename:
		state.defaultFilename = strings.TrimSpace(cmd.restOfCmd)
	case commandGlobal:
		err = cmd.CmdGlobal(state)
	case commandGlobalInteractive:
		fmt.Println("not yet implemented")
	case commandHelp:
		err = cmd.Help(state)
	case commandInverseGlobal:
		fmt.Println("not yet implemented")
	case commandInverseGlobalInteractive:
		fmt.Println("not yet implemented")
	case commandJoin:
		err = cmd.Join(state)
	case commandMark:
		err = cmd.Mark(state)
	case commandList:
		fmt.Println("not yet implemented")
	case commandMove:
		err = cmd.Move(state)
	case commandNumber, commandPrint:
		err = cmd.Print(state)
	case commandPrompt:
		state.ShowPrompt = !state.ShowPrompt
	case commandQuit, commandQuitUnconditionally:
		if cmd.cmd == commandQuit && state.changedSinceLastWrite {
			fmt.Println(unsavedChanges)
		} else {
			quit = true
		}
	case commandRead:
		err = cmd.Read(state)
	case commandSubstitute:
		err = cmd.CmdSubstitute(state)
	case commandTransfer:
		err = cmd.Transfer(state)
	case commandUndo:
		err = cmd.Undo(state)
	case commandWrite:
		err = cmd.Write(state)
		quit = (cmd.cmd == commandWrite && strings.HasPrefix(cmd.restOfCmd, commandQuit))
	case commandWriteAppend:
		fmt.Println("not yet implemented")
	case commandPut:
		err = cmd.Put(state)
	case commandYank:
		err = cmd.Yank(state)
	case commandScroll:
		err = cmd.Scroll(state)
	case commandComment:
		err = cmd.Comment(state)
	case commandLinenumber:
		err = cmd.Linenumber(state)
	case commandNoCommand:
		// nothing entered -- ignore
	default:
		fmt.Println("ERROR got command not in switch!?: ", cmd.cmd)
	}
	return quit, err
}

func minIntOf(vars ...int) int {
	min := vars[0]
	for _, i := range vars {
		if min > i {
			min = i
		}
	}
	return min
}
