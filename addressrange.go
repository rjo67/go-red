package red

import (
	"container/list"
	"errors"
	"fmt"
	"regexp"
	"strings"
	//	"strconv"
)

/*
Names of the capture groups. Must correspond to the regex.
*/
const (
	firstAddressCaptureGroup  string = "address1"
	secondAddressCaptureGroup string = "address2"
	separatorCaptureGroup     string = "separator"

	separatorComma     string = ","
	separatorSemicolon string = ";"
)

var (
	errBadRange                  error = errors.New("address range start > end")
	errInvalidStartOfRange       error = errors.New("invalid start of range")
	errInvalidEndOfRange         error = errors.New("invalid end of range")
	ErrRangeShouldNotBeSpecified error = errors.New("a range may not be specified")
	errUnrecognisedRange         error = errors.New("unrecognised address range")
)

var addressRangeRE = regexp.MustCompile(`^(?P<address1>[^,;]*)` + `(?P<separator>[,;]?)` + `(?P<address2>[^,;]*)$`)

/*
An AddressRange stores the start and end addresses of a range.
*/
type AddressRange struct {
	start, end Address
	separator  string // , or ;
}

func (r AddressRange) String() string {
	return fmt.Sprintf("%v%s%v", r.start, r.separator, r.end)
}

/*
 If an address range has been specified, returns the actual start and end line numbers
  as given by calculateStartAndEndLineNumbers. It is an error if start > end.
 Otherwise, returns the current line number as start and end.
*/
func (ra AddressRange) getAddressRange(currentLineNbr int, buffer *list.List) (startLine int, endLine int, err error) {
	if !ra.IsSpecified() {
		return currentLineNbr, currentLineNbr, nil
	}
	return ra.calculateStartAndEndLineNumbers(currentLineNbr, buffer)
}

/*
 Calculates the start and end line numbers from the given address range.
 It is an error if start > end.
*/
func (ra *AddressRange) calculateStartAndEndLineNumbers(currentLineNbr int, buffer *list.List) (startLine int, endLine int, err error) {
	// special case 1: first address empty -> {1,addr} or {.;addr}
	if ra.start.isNotSpecified() {
		switch ra.separator {
		case separatorComma:
			startLine = 1
		case separatorSemicolon:
			startLine = currentLineNbr
		}
	} else {
		if startLine, err = ra.start.calculateActualLineNumber(currentLineNbr, buffer); err != nil {
			return -1, -1, errInvalidStartOfRange
		}
	}

	// special case 2) first address given, second empty -> {<given address>, <given address>}
	if ra.end.isNotSpecified() {
		endLine = startLine
	} else {
		if endLine, err = ra.end.calculateActualLineNumber(currentLineNbr, buffer); err != nil {
			return -1, -1, errInvalidEndOfRange
		}
	}

	// start must be before end ('special' values excluded)
	if startLine >= 0 && endLine >= 0 && startLine > endLine {
		return -1, -1, errBadRange
	}

	return startLine, endLine, nil
}

/*
 IsSpecified returns TRUE if the given address range contains valid values.
*/
func (ra AddressRange) IsSpecified() bool {
	return !(ra.start.isNotSpecified() && ra.end.isNotSpecified())
}

/*
newRange creates an AddressRange from the given string.

An AddressRange is two addresses separated either by a comma (',') or a semicolon (';').
In a semicolon-delimited range, the current address ('.') is set to the first address before the second address is calculated.
This feature can be used to set the starting line for searches if the second address contains a regular expression.

Addresses can be omitted on either side of the comma or semicolon separator.

The value of the first address in a range cannot exceed the value of the second.

  Special cases:
    empty string        {notSpecified, notSpecified}
    .                   {currentLine, currentLine}
    $                   {endOfFile, endOfFile}
    ,                   {startOfFile, endOfFile}
    n                   {n, n}

  Otherwise, a range in format A1[,;]A2 is expected.
 *
*/
func newRange(rangeStr string) (AddressRange, error) {

	var addrRange AddressRange

	// this case does not seem to be caught by the following switch, therefore handle it specially
	if len(strings.TrimSpace(rangeStr)) == 0 {
		startAddr, err := newAddress(rangeStr)
		if err != nil {
			return addrRange, err
		}
		return AddressRange{startAddr, startAddr, identComma}, nil
	}
	// a few special cases to start with
	switch rangeStr {
	case identDot:
	case identDollar:
		startAddr, err := newAddress(rangeStr)
		if err != nil {
			return addrRange, err
		}
		return AddressRange{startAddr, startAddr, identComma}, nil
	case identComma: // ==(1,$)
		startAddr, err := newAddress("1")
		if err != nil {
			return addrRange, err
		}
		endAddr, err := newAddress(identDollar)
		if err != nil {
			return addrRange, err
		}
		return AddressRange{startAddr, endAddr, identComma}, nil
	case identSemicolon: // ==(.,$)
		startAddr, err := newAddress(identDot)
		if err != nil {
			return addrRange, err
		}
		endAddr, err := newAddress(identDollar)
		if err != nil {
			return addrRange, err
		}
		return AddressRange{startAddr, endAddr, identComma}, nil
	}

	matches := findNamedMatches(addressRangeRE, rangeStr, false)
	if matches == nil {
		return addrRange, errUnrecognisedRange
	}

	start, err := newAddress(matches[firstAddressCaptureGroup])
	if err != nil {
		return addrRange, fmt.Errorf("cannot parse address 1: %s", err.Error())
	}
	end, err := newAddress(matches[secondAddressCaptureGroup])
	if err != nil {
		return addrRange, fmt.Errorf("cannot parse address 2: %s", err.Error())
	}
	separator := matches[separatorCaptureGroup]

	/* TODO

	// special cases: first address empty -> {1,addr} or {.;addr}
	// TODO check if 2nd addr is present
	if start.isNotSpecified() {
		switch separator {
		case separatorComma:
			if start, err = newAddress("1"); err != nil {
				return addrRange, err
			}
		case separatorSemicolon:
			if start, err = newAddress(identDot); err != nil {
				return addrRange, err
			}
		}
	}

	// first address given, second empty -> {<given address>, <given address>}
	if end.isNotSpecified() {
		end = start
	}
	*/

	return AddressRange{start, end, separator}, nil
}

/*
newValidRange is like newRange but panics if the address range cannot be parsed.
Primarily but not solely for test use.
*/
func newValidRange(rangeStr string) AddressRange {
	ra, err := newRange(rangeStr)
	if err != nil {
		panic(fmt.Sprintf("addressrange: cannot parse: %s: %s", rangeStr, err.Error()))
	}
	return ra
}
