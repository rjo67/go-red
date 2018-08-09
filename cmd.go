package main

import (
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
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

var justNumberRE = regexp.MustCompile("^\\s*(\\d*)\\s*$")
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
func CmdAppendInsert(cmd Command, state *State) error {
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
		moveToLine(lineNbr, state)
		mark := state.dotline
		for e := newLines.Front(); e != nil; e = e.Next() {
			mark = state.buffer.InsertAfter(e.Value, mark)
		}
		moveToLine(lineNbr+nbrLinesEntered, state)
	}

	return nil
}

/*
 * Deletes the addressed lines from the buffer.
 *
 * The current address is set to the new address of the line after the last line deleted;
 * if the lines deleted were originally at the end of the buffer,
 * the current address is set to the address of the new last line;
 * if no lines remain in the buffer, the current address is set to zero.
 */
func CmdDelete(cmd Command, state *State) error {
	startLineNbr, endLineNbr, err := calculateStartAndEndLineNumbers(cmd.addrRange, state)
	if err != nil {
		return err
	}
	if startLineNbr == 0 {
		return invalidLine
	}
	moveToLine(startLineNbr, state)

	el := state.dotline
	for lineNbr := startLineNbr; lineNbr <= endLineNbr; lineNbr++ {
		elementToDelete := el
		el = el.Next()
		state.buffer.Remove(elementToDelete)
	}
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
 * Edits file, and sets the default filename.
 * If file is not specified, then the default filename is used.
 * Any lines in the buffer are deleted before the new file is read.
 * The current address is set to the address of the last line in the buffer.
 */
func CmdEdit(cmd Command, state *State) error {
	filename, err := getFilename(strings.TrimSpace(cmd.restOfCmd), state)
	if err != nil {
		return err
	}
	nbrBytesRead, listOfLines, err := ReadFile(filename)
	if err != nil {
		return err
	}
	fmt.Println(nbrBytesRead)
	state.buffer = listOfLines
	state.changedSinceLastWrite = false
	moveToLine(state.buffer.Len(), state)
	return nil
}

/*
 if potentialFilename is set, returns this having set the state.defaultFilename as well.
 Otherwise, returns the state.defaultFilename

 It is an error if both potentialFilename and state.defaultFilename are empty.
 */
func getFilename(potentialFilename string, state *State) (filename string, err error) {
	if potentialFilename == "" {
		// use default if set
		if state.defaultFilename != "" {
			filename = state.defaultFilename
		} else {
			return "", missingFilename
		}
	} else {
		filename = potentialFilename
		state.defaultFilename = potentialFilename
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
func CmdWrite(cmd Command, state *State) error {
	// save current address
	currentLine := state.lineNbr

	// handle command sequence 'wq'
	filename := strings.TrimPrefix(cmd.restOfCmd, commandQuit)
	filename, err := getFilename(strings.TrimSpace(filename), state)
	if err != nil {
		return err
	}

	startLineNbr, endLineNbr, err := calculateStartAndEndLineNumbers(cmd.addrRange, state)
	if err != nil {
		return err
	}
	fmt.Printf("start: %d, end: %d, range: %v\n", startLineNbr, endLineNbr, cmd.addrRange)
	// disallow 0
	if startLineNbr == 0 {
		return invalidLine
	}
	moveToLine(startLineNbr, state)
	nbrBytesWritten, err := WriteFile(filename, state.dotline, startLineNbr, endLineNbr)
	if err != nil {
		return err
	}
	fmt.Println(nbrBytesWritten)
	state.changedSinceLastWrite = false
	moveToLine(currentLine, state)
	return nil
}

/*
 * Prints the addressed lines. The current address is set to the address of the last line printed.
 */
func CmdPrint(cmd Command, state *State) error {
	return _printRange(os.Stdout, cmd, state)
}

/*
 * Prints the addressed lines, preceding each line by its line number and a <tab>.
 * The current address is set to the address of the last line printed.
 */
func CmdNumber(cmd Command, state *State) error {
	return _printRange(os.Stdout, cmd, state)
}

func _printRange(writer io.Writer, cmd Command, state *State) error {
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
		if cmd.cmd == commandNumber {
			fmt.Fprintf(writer, "%4d%c %s", lineNbr, '\t', el.Value.(Line).line)
		} else {
			fmt.Fprintf(writer, el.Value.(Line).line)
		}
		prevEl  = el // store el, to be able to set dotline i/c we hit the end of the list
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
