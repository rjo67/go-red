package red

import (
	"container/list"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// suffixes for the 's' command
const suffixGlobal string = "g" // every match is replaced
const suffixList string = "l"   // list
const suffixNumber string = "n" // number
const suffixPrint string = "p"  // print

var errSyntaxMissingDelimiter error = errors.New("Missing delimiter")
var errNoSubstitutions error = errors.New("No substitution performed")
var errNoPreviousRegex error = errors.New("No previous regex")

/*
CmdGlobal processes the global command, which makes two passes over the file.
 On the first pass, all the addressed lines matching a regular expression re are marked.
 Then, going sequentially from the beginning of the file to the end of the file,
 the given command-list is executed for each marked line,
 with the current address set to the address of that line.
 Any line modified by the command-list is unmarked.

 The final value of the current address is the value assigned by the last command
 in the last command-list executed. If there were no matching lines, the current address is unchanged.

 The first command of command-list must appear on the same line as the 'g' command.
 All lines of a multi-line command-list except the last line must be terminated with a backslash ('\').
 Any commands are allowed, except for 'g', 'G', 'v', and 'V'.
 The '.' terminating the input mode of commands 'a', 'c', and 'i' can be omitted
   if it would be the last line of command-list.
 By default, a newline alone in command-list is equivalent to a 'p' command.
 If ed is invoked with the command-line option '-G', then a newline in command-list
    is equivalent to a '.+1p' command.

 (This is similar to the Substitute command, except the replacement string can be a list of commands)
*/
func (cmd Command) CmdGlobal(state *State) error {
	currentLineNbr := state.lineNbr
	startLineNbr, endLineNbr, err := cmd.AddrRange.getAddressRange(state)
	if err != nil {
		return err
	}

	// find matching lines

	fmt.Println("not yet", startLineNbr, endLineNbr, currentLineNbr)
	return nil
}

/*
CmdSubstitute replaces text in the addressed lines matching a regular expression re with replacement.
 By default, only the first match in each line is replaced.

 The 's' command accepts any combination of the suffixes 'g', 'count', 'l', 'n', and 'p'.
 If the 'g' (global) suffix is given, then every match is replaced.
 The 'count' suffix, where count is a positive number, causes only the countth match to be replaced.
 'g' and 'count' can't be specified in the same command.
 'l', 'n', and 'p' are the usual print suffixes.

 It is an error if no substitutions are performed on any of the addressed lines.
 The current address is set to the address of the last line on which a substitution occurred.
 If a line is split, a substitution is considered to have occurred on each of the new lines.
 If no substitution is performed, the current address is unchanged.

 re and replacement may be delimited by any character other than <space>, <newline>
 and the characters used by the form of the 's' command shown below.
 If the last delimiter is omitted, then the last line affected is printed as if the
 print suffix 'p' were specified. The last delimiter can't be omitted if the 's' command
 is part of a 'g' or 'v' command-list and is not the last command in the list, because
 the meaning of the following escaped newline becomes ambiguous.

 An unescaped '&' in replacement is replaced by the currently matched text.
 The character sequence '\m' where m is a number in the range [1,9], is replaced by the
 mth backreference expression of the matched text. If the corresponding backreference expression
 does not match, then the character sequence '\m' is replaced by the empty string.
 If replacement consists of a single '%', then replacement from the last substitution is used.

 A line can be split by including a newline escaped with a backslash ('\') in replacement,
 except if the 's' command is part of a 'g' or 'v' command-list, because in this case the meaning
 of the escaped newline becomes ambiguous. Each backslash in replacement removes the
 special meaning (if any) of the following character.

 (.,.)s
    Repeats the last substitution.
	 This form of the 's' command accepts the 'g' and 'count' suffixes described above,
	 and any combination of the suffixes 'p' and 'r'.
	 The 'g' suffix toggles the global suffix of the last substitution and resets count to 1.
	 The 'p' suffix toggles the print suffixes of the last substitution.
	 The 'r' suffix causes the re of the last search to be used instead of the re of the last
	 substitution (if the search happened after the substitution).

 Undo is handled by a 'special' internal command 'internalCommandUndoSubst'.
*/
func (cmd Command) CmdSubstitute(state *State) error {

	startLineNbr, endLineNbr, err := cmd.AddrRange.getAddressRange(state)
	if err != nil {
		return err
	}

	var nbrLinesChanged int
	var undoList *list.List
	regexCommand := strings.TrimSpace(cmd.restOfCmd)
	if regexCommand != "" {
		re, replacement, suffixes, err := parseRegexCommand(regexCommand)
		if err != nil {
			return err
		}
		nbrLinesChanged, undoList, err = processLines(os.Stdout, startLineNbr, endLineNbr, state, re, replacement, suffixes)
	} else {
		// TODO need to handle flags on a pure "s" command
		suffixes := strings.TrimSpace(cmd.restOfCmd)
		nbrLinesChanged, undoList, err = processLinesUsingPreviousSubst(os.Stdout, startLineNbr, endLineNbr, state, suffixes)
	}

	if err != nil {
		return err
	}
	if nbrLinesChanged == 0 {
		return errNoSubstitutions
	}

	fmt.Printf("%d lines changed\n", nbrLinesChanged)

	if undoList.Len() != nbrLinesChanged {
		panic(fmt.Sprintf("changed %d lines but undoList contains %d elements", nbrLinesChanged, undoList.Len()))
	}
	state.addUndo(1, 1, internalCommandUndoSubst, undoList, cmd)

	state.changedSinceLastWrite = true
	return nil
}

func parseRegexCommand(regexCommand string) (re, replacement, suffixes string, err error) {
	delimiter := regexCommand[0:1]
	split := strings.Split(regexCommand, delimiter)
	if len(split) != 4 || split[1] == "" {
		return "", "", "", errSyntaxMissingDelimiter
	}
	return split[1], split[2], split[3], nil
}

/*
 Repeats the previous substitution if one is present in state.
 suffixes: gpln or <count> (see doc)

 Returns:
  - number of lines matched
  - a list of undo objects to undo these changes (empty list if no lines changed)
*/
func processLinesUsingPreviousSubst(writer io.Writer, startLineNbr, endLineNbr int,
	state *State, suffixes string) (int, *list.List, error) {
	if state.lastSubstRE != nil {
		// if no suffixes defined, use previously stored
		if suffixes == "" {
			suffixes = state.lastSubstSuffixes
		}
		return replaceLines(writer, startLineNbr, endLineNbr, state, state.lastSubstRE, state.lastSubstReplacement, suffixes)
	}
	return 0, nil, errNoPreviousRegex
}

/*
 Replace lines between start and end matching 'reStr'.
 suffixes: gpln or <count> (see doc)

 Returns:
  - number of lines matched
  - a list of undo objects to undo these changes (empty list if no lines changed)

 Sets state.lastSubstRE, state.lastSubstReplacement, state.lastSubstSuffixes
*/
func processLines(writer io.Writer, startLineNbr, endLineNbr int,
	state *State, reStr, replacement, suffixes string) (int, *list.List, error) {
	re, err := regexp.Compile(reStr)
	if err != nil {
		return 0, nil, err
	}
	state.lastSubstRE = re
	state.lastSubstReplacement = replacement
	state.lastSubstSuffixes = suffixes
	return replaceLines(writer, startLineNbr, endLineNbr, state, re, replacement, suffixes)
}

/*
 Replace lines between start and end matching the given regexp.
 suffixes: gpln or <count> (see doc)

 Returns:
  - number of lines matched
  - a list of undo objects to undo these changes (empty list if no lines changed)
*/
func replaceLines(writer io.Writer, startLineNbr, endLineNbr int,
	state *State, re *regexp.Regexp, replacement, suffixes string) (int, *list.List, error) {

	// evaluate suffixes
	printLineNumbers := strings.Contains(suffixes, suffixNumber)
	printLine := strings.Contains(suffixes, suffixPrint)
	printLineList := strings.Contains(suffixes, suffixList)
	if printLineList {
		fmt.Fprintf(writer, "(suffix %s not supported, defaulting to %s)", suffixList, suffixPrint)
		printLine = true
	}
	//global := strings.Contains(suffixes, suffixGlobal)

	moveToLine(startLineNbr, state)
	nbrLinesMatched := 0
	undoList := list.New()

	el := state.dotline
	for lineNbr := startLineNbr; lineNbr <= endLineNbr; lineNbr++ {
		line := el.Value.(Line)
		if re.MatchString(line.Line) {
			nbrLinesMatched++
			// currently always "global" -- check out ReplaceAllFunc possibly?
			changedLine := re.ReplaceAllString(line.Line, replacement)
			if printLine || printLineNumbers {
				_printLine(writer, lineNbr, changedLine, printLineNumbers)
			}
			el.Value = Line{changedLine}
			// create undo command -- is handled as a 'change' on this line
			currentLine := Address{lineNbr, 0}
			undoCommand := Command{AddressRange{currentLine, currentLine}, commandChange, ""}
			tmpList := list.New()
			tmpList.PushFront(line)
			undoList.PushBack(Undo{undoCommand, tmpList, Command{} /* TODO */})
		}

		el = el.Next()
	}
	return nbrLinesMatched, undoList, nil
}
