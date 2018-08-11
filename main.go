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
	// the last line number is accessible via buffer.Len()
	buffer                *list.List    // the current buffer -- should never be null
	cutBuffer             *list.List   // the cut buffer, set by commands c, d, j, s or y
	dotline               *list.Element // the current (dot) line -- can be null
	lineNbr               int           // the current line number
	changedSinceLastWrite bool          // whether the buffer has been changed since the last write
	defaultFilename       string
	windowSize            int // window size - for scroll command

	// program flags
	debug      bool // debugging activated?
	prompt     string
	showPrompt bool
}

func main() {
	state := State{}
	state.buffer = list.New()
	state.cutBuffer = list.New()

	flag.BoolVar(&state.debug, "d", false, "debug mode")
	// default is set to true
	flag.StringVar(&state.prompt, "p", "", "Specifies a command prompt")
	flag.Parse()

	if state.prompt != "" {
		state.showPrompt = true
	}

	state.windowSize = 15 // see https://stackoverflow.com/a/48610796 for a better way...

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
						err = cmd.CmdAppendInsert(&state)
					case commandChange:
						err = cmd.CmdChange(&state)
					case commandDelete:
						err = cmd.CmdDelete(&state)
					case commandEdit:
						if state.changedSinceLastWrite {
							fmt.Println(unsavedChanges)
						} else {
							err = cmd.CmdEdit(&state)
						}
					case commandEditUnconditionally:
						err = cmd.CmdEdit(&state)
					case commandFilename:
						state.defaultFilename = strings.TrimSpace(cmd.restOfCmd)
					case commandGlobal:
						fmt.Println("not yet implemented")
					case commandGlobalInteractive:
						fmt.Println("not yet implemented")
					case commandJoin:
						err = cmd.CmdJoin(&state)
					case commandMark:
						fmt.Println("not yet implemented")
					case commandList:
						fmt.Println("not yet implemented")
					case commandMove:
						err = cmd.CmdMove(&state)
					case commandNumber:
						err = cmd.CmdNumber(&state)
					case commandPrint:
						err = cmd.CmdPrint(&state)
					case commandPrompt:
						state.showPrompt = !state.showPrompt
					case commandQuit, commandQuitUnconditionally:
						if cmd.cmd == commandQuit && state.changedSinceLastWrite {
							fmt.Println(unsavedChanges)
						} else {
							quit = true
						}
					case commandRead:
						err = cmd.CmdRead(&state)
					case commandSubstitute:
						fmt.Println("not yet implemented")
					case commandTransfer:
						err = cmd.CmdTransfer(&state)
					case commandUndo:
						fmt.Println("not yet implemented")
					case commandInverseGlobal:
						fmt.Println("not yet implemented")
					case commandInverseGlobalInteractive:
						fmt.Println("not yet implemented")
					case commandWrite:
						err = cmd.CmdWrite(&state)
						quit = (cmd.cmd == commandWrite && strings.HasPrefix(cmd.restOfCmd, commandQuit))
					case commandWriteAppend:
						fmt.Println("not yet implemented")
					case commandPut:
						err = cmd.CmdPut(&state)
					case commandYank:
						err = cmd.CmdYank(&state)
					case commandScroll:
						err = cmd.CmdScroll(&state)
					case commandComment:
						fmt.Println("not yet implemented")
					case commandLinenumber:
						err = cmd.CmdLinenumber(&state)
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
					fmt.Printf("state: %+v, buffer len: %d, cut buffer len %d\n", state, state.buffer.Len(), state.cutBuffer.Len())
				}
			}
		}
	}
}
