package main

import (
	"bufio"
	"flag"
	"fmt"
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

	if state.Prompt == "" {
		state.Prompt = ":" // default prompt
	}
	state.ShowPrompt = true

	state.WindowSize = 15 // see https://stackoverflow.com/a/48610796 for a better way...

	fmt.Printf("*** %s (v%s)\n", NAME, VERSION)
	mainloop(state)
}

func mainloop(state *red.State) {
	reader := bufio.NewReader(os.Stdin)
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
			fmt.Printf("error: %s", err)
		} else {
			cmd, err := red.ParseCommand(cmdStr)
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
