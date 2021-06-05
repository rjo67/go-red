package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/rjo67/red"
)

/*
VERSION is the program version
*/
const VERSION = "0.4"

/*
NAME is the progam name
*/
const NAME = "Rich's ed"

func main() {
	state := red.NewState()

	flag.BoolVar(&state.Debug, "d", false, "debug mode")
	flag.BoolVar(&state.ShowMemory, "m", false, "show memory usage")
	flag.StringVar(&state.Prompt, "p", "", "Specifies a command prompt (default ':')")
	flag.Parse()

	stop := false
	var startfile string
	if flag.NArg() > 1 {
		fmt.Printf("unexpected arguments. See usage")
		stop = true
	} else if flag.NArg() == 1 {
		startfile = flag.Arg(0)
	}
	if !stop {
		if state.Prompt == "" {
			state.Prompt = ":" // default prompt
		}
		state.ShowPrompt = true

		state.WindowSize = 15 // see https://stackoverflow.com/a/48610796 for a better way...

		fmt.Printf("*** %s (v%s)\n", NAME, VERSION)
	}
	if !stop {
		// read in start file if specified
		if startfile != "" {
			if err := readInputFile(startfile, state); err != nil {
				fmt.Printf("error: %s\n", err.Error())
				stop = true
			}
		}
	}
	if !stop {
		mainloop(state, bufio.NewReader(os.Stdin))
	}
}

/*
Reads the given file into 'state'.
*/
func readInputFile(filename string, state *red.State) error {
	// read in file
	editCommandStr := "e " + filename
	cmd, err := red.ParseCommand(editCommandStr, state.Debug)
	if err != nil {
		return fmt.Errorf("could not parse command %s", editCommandStr)
	} else {
		err = cmd.Edit(state)
		if err != nil {
			return fmt.Errorf("error reading file %s", filename)
		}
	}
	return nil
}

func mainloop(state *red.State, reader *bufio.Reader) {
	quit := false
	for !quit {
		if state.ShowMemory {
			fmt.Printf("%s ", GetMemUsage())
		}
		if state.ShowPrompt {
			fmt.Print(state.Prompt, " ")
		}
		cmdStr, err := reader.ReadString('\n')
		if err != nil {
			// EOF might happen if reading commands from input file
			if err == io.EOF {
				quit = true
			} else {
				fmt.Printf("error: %s", err)
			}
		} else {
			cmd, err := red.ParseCommand(cmdStr[0:len(cmdStr)-1], state.Debug) // remove LF
			if err != nil {
				fmt.Printf("? %s\n", err)
			} else {
				if state.Debug {
					fmt.Println(cmd)
				}

				var err error
				quit, err = cmd.ProcessCommand(state, nil, false)

				// each command call can return an error, which will be displayed here
				if err != nil {
					fmt.Printf("error: %s\n", err)
				}
				if state.Debug {
					fmt.Printf("state: %+v, buffer len: %d, cut buffer len %d\n", state, state.Buffer.Len(), state.CutBuffer.Len())
				}
			}
		}
	}
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
