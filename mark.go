package red

import "fmt"

/**
addMark adds the given mark to the list of marks.
A pre-existing mark with the same name will be replaced.
*/
func (state *State) addMark(name string, lineNbr int) {
	state.marks[name] = lineNbr
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
		for markName, lineNbr := range state.marks {
			if lineNbr <= endLine && lineNbr >= startLine {
				delete(state.marks, markName)
			} else if lineNbr > endLine {
				state.marks[markName] = lineNbr - nbrLinesDeleted
			}
		}
	case commandMove:
		nbrLinesMoved := endLine - startLine + 1
		movingFromAbove := startLine > destination
		for markName, lineNbr := range state.marks {
			// marks in the range should be removed
			// marks between range and destination should be reduced
			// range below destination: marks above destination should be increased
			// marks above destination not affected (if range came from above destination)
			switch movingFromAbove {
			case true:
				if lineNbr > endLine || lineNbr <= destination {
					// no-op
				} else if lineNbr <= endLine && lineNbr >= startLine {
					delete(state.marks, markName)
				} else {
					state.marks[markName] = lineNbr + nbrLinesMoved
				}
			case false:
				if lineNbr < startLine || lineNbr > destination {
					// no-op
				} else if lineNbr <= endLine && lineNbr >= startLine {
					delete(state.marks, markName)
				} else {
					state.marks[markName] = lineNbr - nbrLinesMoved
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
