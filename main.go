package main

import (
	"bufio"
	"container/list"
	"flag"
	"fmt"
	"os"
	"strings"
)

const unsavedChanges string = "buffer has unsaved changes"
/**
 * Stores information about a line.
 * The line number is not stored, this is implicit.
 */
type Line struct {
	line string
}

type State struct {
	// the current buffer -- should never be null
	// the last line number is accessible via buffer.Len()
	buffer *list.List
	// the current (dot) line -- can be null
	dotline *list.Element
	// the current line number
	lineNbr int
	// whether the buffer has been changed since the last write
	changedSinceLastWrite bool
	defaultFilename       string

	// program flags
	debug      bool // debugging activated?
	prompt     string
	showPrompt bool
}

func main() {
	state := State{}
	state.buffer = list.New()

	flag.BoolVar(&state.debug, "d", false, "debug mode")
	// default is set to true
	flag.StringVar(&state.prompt, "p", "", "Specifies a command prompt")
	flag.Parse()

	if state.prompt != "" {
		state.showPrompt = true
	}

	mainloop(state)
}

func mainloop(state State) {
	reader := bufio.NewReader(os.Stdin)
	quit := false
	for !quit {
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
					commandFilename, commandPrompt, commandQuit, commandQuitUnconditionally,
					commandUndo:
					if cmd.addrRange.isAddressRangeSpecified() {
						err = rangeShouldNotBeSpecified
					}
				default:
					//ok
				}
				if err == nil {
					switch cmd.cmd {
					case commandAppend, commandInsert:
						err = CmdAppendInsert(cmd, &state)
					case commandChange:
						fmt.Println("not yet implemented")
					case commandDelete:
						err = CmdDelete(cmd, &state)
					case commandEdit:
						if state.changedSinceLastWrite {
							fmt.Println(unsavedChanges)
						} else {
							err = CmdEdit(cmd, &state)
						}
					case commandEditUnconditionally:
						err = CmdEdit(cmd, &state)
					case commandFilename:
						state.defaultFilename = strings.TrimSpace(cmd.restOfCmd)
					case commandGlobal:
						fmt.Println("not yet implemented")
					case commandGlobalInteractive:
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
						err = CmdNumber(cmd, &state)
					case commandPrint:
						err = CmdPrint(cmd, &state)
					case commandPrompt:
						state.showPrompt = !state.showPrompt
					case commandQuit, commandQuitUnconditionally:
						if cmd.cmd == commandQuit && state.changedSinceLastWrite {
							fmt.Println(unsavedChanges)
						} else {
							quit = true
						}
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
						err = CmdWrite(cmd, &state)
						quit = (cmd.cmd == commandWrite && strings.HasPrefix(cmd.restOfCmd, commandQuit))
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
					case commandNoCommand:
						// nothing entered -- ignore
					default:
						fmt.Println("ERROR got command not in switch!?")
					}
				}
				// each command call can return an error, which will be displayed here
				if err != nil {
					fmt.Printf("error: %s\n", err)
				}
				if state.debug {
					fmt.Printf("state: %+v, buffer length: %d\n", state, state.buffer.Len())
				}
			}
		}
	}
}
