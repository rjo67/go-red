package main

import (
	"bufio"
	"container/list"
	"flag"
	"fmt"
	"os"
)

/**
 * Stores information about a line.
 * The line number is not stored, this is implicit.
 */
type Line struct {
	line string
}

type State struct {
	// the current buffer -- should never be null
	buffer *list.List
	// the current (dot) line -- can be null
	dotline *list.Element
	// the current line number
	lineNbr int
	// the last line number
	lastLineNbr int
	// whether the buffer has been changed since the last write
	changedSinceLastWrite bool

	// program flags
	debug bool // debugging activated?
}

func main() {
	state := State{}
	state.buffer = list.New()

	flag.BoolVar(&state.debug, "d", false, "activate debug mode")
	flag.Parse()

	mainloop(state)
}

func mainloop(state State) {
	reader := bufio.NewReader(os.Stdin)
	quit := false
	for !quit {
		cmdStr, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("error: %s", err)
		} else {
			cmd, err := ParseCommand(cmdStr)
			if err != nil {
				fmt.Printf("? %s\n", err)
			} else {
				if state.debug {
					fmt.Println(cmd)
				}
				var err error
				switch cmd.cmd {
				case commandAppend:
					err = CmdAppend(cmd, &state)
				case commandChange:
					fmt.Println("not yet implemented")
				case commandDelete:
					fmt.Println("not yet implemented")
				case commandEdit:
					fmt.Println("not yet implemented")
				case commandEditUnconditionally:
					fmt.Println("not yet implemented")
				case commandFilename:
					fmt.Println("not yet implemented")
				case commandGlobal:
					fmt.Println("not yet implemented")
				case commandGlobalInteractive:
					fmt.Println("not yet implemented")
				case commandInsert:
					fmt.Println("not yet implemented")
				case commandJoin:
					fmt.Println("not yet implemented")
				case commandMark:
					fmt.Println("not yet implemented")
				case commandList:
					fmt.Println("not yet implemented")
				case commandMove:
					fmt.Println("not yet implemented")
				case commandNumber:
					fmt.Println("not yet implemented")
				case commandPrint:
					err = CmdPrint(cmd, &state)
				case commandQuit:
					if state.changedSinceLastWrite {
						fmt.Println("buffer has unsaved changes")
					} else {
						quit = true
					}
				case commandQuitUnconditionally:
					quit = true
				case commandRead:
					fmt.Println("not yet implemented")
				case commandSubstitute:
					fmt.Println("not yet implemented")
				case commandTransfer:
					fmt.Println("not yet implemented")
				case commandUndo:
					fmt.Println("not yet implemented")
				case commandInverseGlobal:
					fmt.Println("not yet implemented")
				case commandInverseGlobalInteractive:
					fmt.Println("not yet implemented")
				case commandWrite:
					fmt.Println("not yet implemented")
				case commandWriteAppend:
					fmt.Println("not yet implemented")
				case commandPut:
					fmt.Println("not yet implemented")
				case commandYank:
					fmt.Println("not yet implemented")
				case commandScroll:
					fmt.Println("not yet implemented")
				case commandComment:
					fmt.Println("not yet implemented")
				default:
					fmt.Println("ERROR got command not in switch!?")
				}
				// each command call can return an error, which will be displayed here
				if err != nil {
					fmt.Printf("error: %s\n", err)
				}
				if state.debug {
					fmt.Printf("%+v\n", state)
				}
			}
		}
	}
}

func calculateLastLineNbr(state *State) int {
	if state.buffer == nil {
		return 0
	}
	lineNbr := 0
	for e := state.buffer.Front(); e != nil; e = e.Next() {
		lineNbr++
	}
	return lineNbr
}
