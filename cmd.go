package main

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"os"
	"regexp"
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

type CommandError struct {
	desc string
}

func (e *CommandError) Error() string {
	return e.desc
}

var unrecognisedCommand CommandError = CommandError{"unrecognised command"}
var rangeShouldNotBePresent CommandError = CommandError{"a range should not be specified"}

var commandLineRE = regexp.MustCompile("(.*)([acdeEfgGijklmnpqQrstuvVwWxyz#=])(.*)")

/*
 * A command, parsed from user input
 */
type Command struct {
	addrRange AddressRange
	cmd       string
	restOfCmd string
}

func ParseCommand(str string) (cmd Command, err error) {
	matches := commandLineRE.FindStringSubmatch(str)
	if matches != nil && len(matches) == 4 {
		addrString := matches[1]
		cmdString := matches[2]
		restOfCmd := matches[3]

		addrRange, err := newRange(addrString)
		if err != nil {
			return Command{}, err
		} else {
			return Command{addrRange, cmdString, restOfCmd}, nil
		}
	} else {
		return Command{}, &unrecognisedCommand
	}
}

/*
 * Appends text to the buffer after the addressed line.
 * The address '0' (zero) is valid for this command; it places the entered text
 * at the beginning of the buffer.
 * Text is entered in input mode.
 * The current address is set to the address of the last line entered or,
 * if there were none, to the addressed line.
 */
func CmdAppend(cmd Command, state *State) error {
	// calc required line (checks if this is a valid line)
	lineNbr, err := calculateActualLineNumber(cmd.addrRange.start, state)
	if err != nil {
		return err
	}

	newLines := list.New()
	reader := bufio.NewReader(os.Stdin)
	nbrLinesEntered := 0
	for quit := false; !quit; {
		inputStr, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("error: %s", err)
		}
		if inputStr == ".\n" {
			quit = true
		} else {
			nbrLinesEntered++
			newLines.PushBack(Line{inputStr})
		}
	}
	// now add 'newLines' list to buffer
	if lineNbr == 0 {
		state.buffer.PushFrontList(newLines)
		moveToLine(nbrLinesEntered, state)
	} else {
		moveToLine(lineNbr, state)
		mark := state.dotline
		for e := newLines.Front(); e != nil; e = e.Next() {
			mark = state.buffer.InsertAfter(e.Value, mark)
		}
		state.dotline = mark
	}
	state.lastLineNbr += nbrLinesEntered
	return nil
}

/*
 * Prints the addressed lines. The current address is set to the address of the last line printed.
 */
func CmdPrint(cmd Command, state *State) error {
	_printRange(os.Stdout, cmd, state)
	return nil
}

func _printRange(writer io.Writer, cmd Command, state *State) error {
	startLineNbr, err := calculateActualLineNumber(cmd.addrRange.start, state)
	if err != nil {
		return err
	}
	if startLineNbr == 0 {
		startLineNbr = 1
	}
	endLineNbr, err := calculateActualLineNumber(cmd.addrRange.end, state)
	if err != nil {
		return err
	}
	if endLineNbr == 0 {
		endLineNbr = 1
	}

	if startLineNbr > endLineNbr {
		panic(fmt.Sprintf("start line: %d, end line %d", startLineNbr, endLineNbr))
	}
	moveToLine(startLineNbr, state)

	el := state.dotline
	for lineNbr := startLineNbr; lineNbr <= endLineNbr; lineNbr++ {
		fmt.Fprintf(writer, el.Value.(Line).line)
		el = el.Next()
	}
	// set dotline
	state.dotline = el
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
