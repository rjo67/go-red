package main

import (
	"bufio"
	"container/list"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

/*
VERSION is the program version
*/
const VERSION = "0.3"

/*
NAME is the progam name
*/
const NAME = "Rich's ed"

const unsavedChanges string = "buffer has unsaved changes"

/*
A Line stores information about a line.
  The line number is not stored, this is implicit.
*/
type Line struct {
	line string
}

func main() {
	state := newState()

	flag.BoolVar(&state.debug, "d", false, "debug mode")
	flag.BoolVar(&state.showMemory, "m", false, "show memory usage")
	flag.StringVar(&state.prompt, "p", "", "Specifies a command prompt (default ':')")
	flag.Parse()

	if state.prompt == "" {
		state.prompt = ":" // default prompt
	}
	state.showPrompt = true

	state.windowSize = 15 // see https://stackoverflow.com/a/48610796 for a better way...

	mainloop(state)
}

func mainloop(state *State) {
	reader := bufio.NewReader(os.Stdin)
	quit := false
	for !quit {
		if state.showMemory {
			fmt.Printf("%s ", GetMemUsage())
		}
		if state.showPrompt {
			fmt.Print(state.prompt, " ")
		}
		cmdStr, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("error: %s", err)
		} else {
			cmd, err := ParseCommand(cmdStr)
			if err != nil {
				fmt.Printf("? %s\n", err)
			} else {
				//if state.debug {
				//	fmt.Println(cmd)
				//}

				var err error

				// first check for commands which cannot take ranges
				switch cmd.cmd {
				case commandEdit, commandEditUnconditionally,
					commandFilename, commandHelp, commandPrompt,
					commandQuit, commandQuitUnconditionally,
					commandUndo:
					if cmd.addrRange.isAddressRangeSpecified() {
						err = errRangeShouldNotBeSpecified
					}
				default:
					//ok
				}
				if err == nil {
					quit, err = processCommand(cmd, state, nil, false)
				}
				// each command call can return an error, which will be displayed here
				if err != nil {
					fmt.Printf("error: %s\n", err)
				}
				if state.debug {
					fmt.Printf("state: %+v, buffer len: %d, cut buffer len %d\n", state, state.buffer.Len(), state.cutBuffer.Len())
				}
			}
		}
	}
}

/*
 Processes the given command.

 enteredText is non-nil if we're processing an undo (e.g. undoing a delete)
 inGlobalCommand is set TRUE if we're already processing a 'g' command,
    in which case certain other commands are not allowed/do not make sense.

 Returns TRUE if the quit command has been given.
*/
func processCommand(cmd Command, state *State, enteredText *list.List, inGlobalCommand bool) (quit bool, err error) {
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
	switch cmd.cmd {
	case commandAppend, commandInsert:
		err = cmd.CmdAppendInsert(state, enteredText)
	case commandChange:
		err = cmd.CmdChange(state, enteredText)
	case commandDelete:
		err = cmd.CmdDelete(state, true)
	case commandEdit:
		if state.changedSinceLastWrite {
			fmt.Println(unsavedChanges)
		} else {
			err = cmd.CmdEdit(state)
		}
	case commandEditUnconditionally:
		err = cmd.CmdEdit(state)
	case commandFilename:
		state.defaultFilename = strings.TrimSpace(cmd.restOfCmd)
	case commandGlobal:
		err = cmd.CmdGlobal(state)
	case commandGlobalInteractive:
		fmt.Println("not yet implemented")
	case commandHelp:
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
	case commandPrompt:
		state.showPrompt = !state.showPrompt
	case commandQuit, commandQuitUnconditionally:
		if cmd.cmd == commandQuit && state.changedSinceLastWrite {
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
	case commandUndo:
		err = cmd.CmdUndo(state)
	case commandWrite:
		err = cmd.CmdWrite(state)
		quit = (cmd.cmd == commandWrite && strings.HasPrefix(cmd.restOfCmd, commandQuit))
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
		fmt.Println("ERROR got command not in switch!?: ", cmd.cmd)
	}
	return quit, err
}

// GetMemUsage returns a formatted string of current memory stats
// from https://golangcode.com/print-the-current-memory-usage/
func GetMemUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	nbrGC := m.NumGC
	gcStr := ""
	if nbrGC > 0 {
		lastGC := time.Unix(0, int64(m.LastGC))
		gcStr = fmt.Sprintf(", GC(#%d @ %s)", nbrGC, lastGC.Format(time.Kitchen))
	}
	return fmt.Sprintf("Heap=%v MiB, Sys=%v MiB%s", bToMb(m.HeapAlloc), bToMb(m.Sys), gcStr)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
