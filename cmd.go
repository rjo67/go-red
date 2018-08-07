package main

import (
	"container/list"
	"fmt"
	"io"
	"os"
)

// ---- constants for the available commands
const commandPrint rune = 'p'
const commandQuit rune = 'q'

/*
 * A command, parsed from user input
 */
type Command struct {
	addrRange AddressRange
	cmd       rune
}

func ParseCommand(str string) (cmd Command, err error) {
	// parse the input until we hit a command or EOL
	// what comes before must be a valid range
	foundCommand := false
	for pos, char := range str {
		//fmt.Printf("pos %d, char %c\n", pos, char)
		switch char {
		case commandPrint:
			addrRange, err := newRange(str[:pos])
			if err != nil {
				return cmd, err
			}
			cmd = Command{addrRange, char}
			foundCommand = true
		case commandQuit:
			cmd = Command{AddressRange{}, char}
			foundCommand = true
		}
	}
	if foundCommand {
		return cmd, nil
	} else {
		return Command{}, nil
	}
}

func CmdPrint(lineList *list.List) {
	_print(os.Stdout, lineList)
}

func _printRange(writer io.Writer, lineList *list.List, addrRange AddressRange) error {
	el, err := moveToLine(lineList, addrRange.start)
	if err != nil {
		return err
	}
	lineNbr := addrRange.start.addr - 1
	for e := el; e != nil; e = e.Next() {
		lineNbr++
		fmt.Fprintf(writer, e.Value.(Line).line)
		if lineNbr == addrRange.end.addr {
			break
		}
	}
	// set dotline
	return nil
}

func _print(writer io.Writer, lineList *list.List) {
	if dotline != nil {
		line := dotline.Value.(Line)
		fmt.Fprintf(writer, line.line)
	}
}

func moveToLine(lineList *list.List, requiredLine Address) (*list.Element, error) {
	lineNbr := 1
	e := lineList.Front()
	//fmt.Printf("line %d, e = %s", lineNbr, e.Value.(Line).line)
	for ; e != nil && requiredLine.addr != lineNbr; lineNbr, e = lineNbr+1, e.Next() {
	}
	if e != nil {
		return e, nil
	} else {
		return nil, &invalidLine
	}
}
