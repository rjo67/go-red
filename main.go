package main

import (
	"bufio"
	"container/list"
	"fmt"
	"os"
)

const badInput string = "?"

/**
 * Stores information about a line.
 * The line number is not stored, this is implicit.
 */
type Line struct {
	line string
}

// the current (dot) line
var dotline *list.Element

func main() {
	fmt.Println("hello")
	args := os.Args[1:]

	fmt.Println(args)

	mainloop()
}

func mainloop() {
	reader := bufio.NewReader(os.Stdin)
	quit := false
	for !quit {
		cmdStr, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("error: %s", err)
		} else {
			//fmt.Printf("got cmd: %s", cmdStr)
			cmd, err := ParseCommand(cmdStr)
			if err != nil {
				fmt.Printf("error: %s\n", err)
			} else {
				fmt.Println(cmd)
			}
			if cmd.cmd == 'q' {
				quit = true
			}
		}
	}
}
