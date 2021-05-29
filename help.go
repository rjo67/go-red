package red

import (
	"fmt"
	"strings"
)

/*
Help displays a list of the available commands.

In the original 'ed' the help command 'explained the last error'.
Here, it prints the list of available commands, or if a command is included (e.g. "h a") then it prints a help for that command.
*/
func (cmd Command) Help(state *State) error {
	fmt.Println()
	if subcmd := strings.TrimSpace(cmd.restOfCmd); len(subcmd) != 0 {
		switch subcmd {
		case commandAppend:
			fmt.Println(" ", commandAppend, "Appends text after the addressed line.")
			fmt.Println("\n  Text is entered in input mode, i.e. any number of lines, terminated by a fullstop on its own line.")
			fmt.Println("  Specifying the address '0' (zero) adds the entered text at the beginning of the buffer.")
			fmt.Println("\n  Ex.: 2a      appends text after line 2.")
		case commandChange:
			fmt.Println(" ", commandChange, "Changes lines in the buffer.")
			fmt.Println("\n  Ex.: 2-4c      changes lines 2-4.")
		case commandDelete:
			fmt.Println(" ", commandDelete, "Deletes lines from the buffer.")
		case commandEdit, commandEditUnconditionally:
			fmt.Println(" ", commandEdit, "Edits (reads in) file, if there are no current unsaved changes.")
			fmt.Println(" ", commandEditUnconditionally, "Edits (reads in) file regardless of any currently unsaved changes.")
		case commandFilename:
			fmt.Println(" ", commandFilename, "Sets the default filename.")
		case commandGlobal, commandGlobalInteractive, commandInverseGlobal, commandInverseGlobalInteractive:
			fmt.Println(" ", commandGlobal, "Executes the command-list for all matching lines.")
			fmt.Println(" ", commandGlobalInteractive, "Interactive 'global'.")
			fmt.Println(" ", commandInverseGlobal, "As 'global' but acts on all lines NOT matching the regex.")
			fmt.Println(" ", commandInverseGlobalInteractive, "Interactive 'inverse-global'.")
		case commandHelp:
			fmt.Println(" ", commandHelp, "Displays this help")
		case commandInsert:
			fmt.Println(" ", commandInsert, "Inserts text before the addressed line.")
			fmt.Println("\n  Text is entered in input mode, i.e. any number of lines, terminated by a fullstop on its own line.")
			fmt.Println("  Specifying the address '0' (zero) adds the entered text at the beginning of the buffer.")
		case commandJoin:
			fmt.Println(" ", commandJoin, "Joins the addressed lines, replacing them by a single line containing the joined text.")
			fmt.Printf("\n  Example: 2,4%s will replace the contents of line 2 with the text of lines 2-4.\n", commandJoin)
			fmt.Println("  (Newlines are replaced by spaces)")
		case commandMark:
			fmt.Println(" ", commandMark, "Marks the given line.")
			fmt.Println("\n  The mark 'a' can be referred to in an address using the syntax 'a.")
		case commandMove:
			fmt.Println(" ", commandMove, "Moves lines in the buffer.")
			fmt.Println("\n  The addressed lines are moved to after the destination address.")
			fmt.Println("  Specifying the destination address '0' (zero) moves the addressed lines to the beginning of the buffer.")
			fmt.Printf("\n  Example: 2,4%s5 moves lines 2-4  to after line 5.\n", commandMove)
		case commandList, commandNumber, commandPrint:
			fmt.Println(" ", commandList, "Display the addressed lines.")
			fmt.Println(" ", commandNumber, "Prints the addressed lines with their line numbers.")
			fmt.Println(" ", commandPrint, "Prints the addressed lines.")
		case commandPrompt:
			fmt.Println(" ", commandPrompt, "Sets the prompt.")
		case commandQuit, commandQuitUnconditionally:
			fmt.Println(" ", commandQuit, "Quits the editor if there are no unsaved changes.")
			fmt.Println(" ", commandQuitUnconditionally, "Quits the editor without saving.")
		case commandRead:
			fmt.Println(" ", commandRead, "Reads a file and appends it after the addressed line.")
			fmt.Println("\n  Specifying the address '0' (zero) adds the file's contents at the beginning of the buffer.")
		case commandSubstitute:
			fmt.Println(" ", commandSubstitute, "Replaces text in lines matching a regular expression.")
			fmt.Println("\n  Allowed suffixes are: 'g' global, 'count', or 'l', 'n', or 'p'.")
			fmt.Println("  The 'count' suffix causes only the 'count'th match to be replaced.")
			fmt.Printf("\n  Example: 2,4%s/re/replacement/g replaces all matches of regex 're' with 'replacement' in lines 2-4.\n", commandSubstitute)
		case commandTransfer:
			fmt.Println(" ", commandTransfer, "Copies (transfers) lines to a destination address.")
		case commandUndo:
			fmt.Println(" ", commandUndo, "Undoes the effect of the last command that modified anything in the buffer.")
		case commandWrite, commandWriteAppend:
			fmt.Println(" ", commandWrite, "Writes the addressed lines to a file.")
			fmt.Println(" ", commandWriteAppend, "Appends the addressed lines to a file.")
		case commandPut, commandYank:
			fmt.Println(" ", commandPut, "Puts (inserts) the cut-buffer after the addressed line.")
			fmt.Println(" ", commandYank, "Copies (yanks) the addressed lines to the cut-buffer.")
		case commandScroll:
			fmt.Println(" ", commandScroll, "Scrolls n lines starting at the addressed line.")
		case commandComment:
			fmt.Println(" ", commandComment, "Enters a comment (i.e. the line is ignored)")
		case commandLinenumber:
			fmt.Println(" ", commandLinenumber, "Prints the line number of the addressed line.")
		default:
			return fmt.Errorf("Command '%s' not recognised. Enter '%s' for a list of all commands", subcmd, commandHelp)
		}
	} else {
		fmt.Println(" ", commandAppend, "Appends text after the addressed line.")
		fmt.Println(" ", commandChange, "Changes lines in the buffer.")
		fmt.Println(" ", commandDelete, "Deletes lines from the buffer.")
		fmt.Println(" ", commandEdit, "Edits (reads in) file, if there are no current unsaved changes.")
		fmt.Println(" ", commandEditUnconditionally, "Edits (reads in) file regardless of any currently unsaved changes.")
		fmt.Println(" ", commandFilename, "Sets the default filename.")
		fmt.Println(" ", commandGlobal, "Executes the command-list for all matching lines.")
		fmt.Println(" ", commandGlobalInteractive, "Interactive 'global'.")
		fmt.Println(" ", commandHelp, "Displays this help. (Specify another command to get help on that command)")
		fmt.Println(" ", commandInsert, "Inserts text before the addressed line.")
		fmt.Println(" ", commandJoin, "Joins the addressed lines, replacing them by a single line containing the joined text.")
		fmt.Println(" ", commandMark, "Marks the given line.")
		fmt.Println(" ", commandList, "Display the addressed lines.")
		fmt.Println(" ", commandMove, "Moves lines in the buffer.")
		fmt.Println(" ", commandNumber, "Prints the addressed lines with their line numbers.")
		fmt.Println(" ", commandPrint, "Prints the addressed lines.")
		fmt.Println(" ", commandPrompt, "Sets the prompt.")
		fmt.Println(" ", commandQuit, "Quits the editor if there are no unsaved changes.")
		fmt.Println(" ", commandQuitUnconditionally, "Quits the editor without saving changes.")
		fmt.Println(" ", commandRead, "Reads file and appends it after the addressed line.")
		fmt.Println(" ", commandSubstitute, "Replaces text in lines matching a regular expression.")
		fmt.Println(" ", commandTransfer, "Copies (transfers) lines to a destination address.")
		fmt.Println(" ", commandUndo, "Undoes the effect of the last command that modified anything in the buffer.")
		fmt.Println(" ", commandInverseGlobal, "As 'global' but acts on all lines NOT matching the regex.")
		fmt.Println(" ", commandInverseGlobalInteractive, "Interactive 'inverse-global'.")
		fmt.Println(" ", commandWrite, "Writes the addressed lines to a file.")
		fmt.Println(" ", commandWriteAppend, "Appends the addressed lines to a file.")
		fmt.Println(" ", commandPut, "Puts (inserts) the cut-buffer after the addressed line.")
		fmt.Println(" ", commandYank, "Copies (yanks) lines to the cut-buffer.")
		fmt.Println(" ", commandScroll, "Scrolls n lines starting at the addressed line.")
		fmt.Println(" ", commandComment, "Enters a comment (i.e. the line is ignored)")
		fmt.Println(" ", commandLinenumber, "Prints the line number of the addressed line.")
	}
	fmt.Println()
	return nil
}
