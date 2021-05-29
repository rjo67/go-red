package red

import "fmt"

/*
Mark stores info about a mark.
*/
type Mark struct {
	name    string // name of the mark
	lineNbr int    // line number
}

/**
addMark adds the given mark to the list of marks.
A pre-existing mark with the same name will be replaced.
*/
func (state *State) addMark(mark Mark) {
	state.marks[mark.name] = mark
}

// updateMarks updates the line numbers of marks after various operations
// destination only relevant for 'move'
func (state *State) updateMarks(cmdIdent string, startLine, endLine, destination int) error {
	if startLine > endLine {
		return fmt.Errorf("updateMarks: bad line numbers: start: %d, end: %d", startLine, endLine)
	}
	switch cmdIdent {
	case commandDelete:
		// after lines have been deleted, any marks in the range should be removed
		// and any marks 'above' the range should be reducted to reflect the new line numbers
		nbrLinesDeleted := endLine - startLine + 1
		for _, mark := range state.marks {
			if mark.lineNbr <= endLine && mark.lineNbr >= startLine {
				delete(state.marks, mark.name)
			} else if mark.lineNbr > endLine {
				state.marks[mark.name] = Mark{name: mark.name, lineNbr: mark.lineNbr - nbrLinesDeleted}
			}
		}
	case commandMove:
		nbrLinesMoved := endLine - startLine + 1
		movingFromAbove := startLine > destination
		for _, mark := range state.marks {
			// marks in the range should be removed
			// marks between range and destination should be reduced
			// range below destination: marks above destination should be increased
			// marks above destination not affected (if range came from above destination)
			switch movingFromAbove {
			case true:
				if mark.lineNbr > endLine || mark.lineNbr <= destination {
					// no-op
				} else if mark.lineNbr <= endLine && mark.lineNbr >= startLine {
					delete(state.marks, mark.name)
				} else {
					state.marks[mark.name] = Mark{name: mark.name, lineNbr: mark.lineNbr + nbrLinesMoved}
				}
			case false:
				if mark.lineNbr < startLine || mark.lineNbr > destination {
					// no-op
				} else if mark.lineNbr <= endLine && mark.lineNbr >= startLine {
					delete(state.marks, mark.name)
				} else {
					state.marks[mark.name] = Mark{name: mark.name, lineNbr: mark.lineNbr - nbrLinesMoved}
				}
			}
			/*
				switch {
				case (!movingFromAbove && mark.lineNbr < startLine)|| (movingFromAbove && mark.lineNbr > endLine):
					// no-op
				case mark.lineNbr <= destination:
					if !movingFromAbove {
						state.marks[mark.name] = Mark{name: mark.name, lineNbr: mark.lineNbr - nbrLinesMoved}
					}
				case mark.lineNbr > destination:
					if !movingFromAbove {
						state.marks[mark.name] = Mark{name: mark.name, lineNbr: mark.lineNbr - nbrLinesMoved}
					}
				default:
					return fmt.Errorf("updateMarks: hit default in switch for mark: %v", mark)
				}
			*/
			/*
				} else if mark.lineNbr > destination {
					state.marks[mark.name] = Mark{name: mark.name, lineNbr: mark.lineNbr + nbrLinesMoved}
				}
			*/
		}
	default:
		return fmt.Errorf("updateMarks: unrecognised command identifier: '%s'", cmdIdent)
	}
	return nil
}
