package red

import (
	"container/list"
	"fmt"
)

/*
Mark stores info about a mark. Namely, a pointer to the appropriate line, and its name.
*/
type Mark struct {
	line *list.Element // pointer to line
	name string        // name of the mark
}

/**
Adds the given mark to the list of marks.
A pre-existing mark with the same name will be replaced.
*/
func (state *State) addMark(mark Mark) {
	state.marks[mark.name] = mark
	fmt.Printf("added mark '%s'\n", mark.name)
}
