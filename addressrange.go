package red

import (
	"errors"
	"fmt"
	"regexp"
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

var _ = fmt.Printf // For debugging; delete when done.

var errInvalidLine error = errors.New("invalid line in address range")
var errUnrecognisedRange error = errors.New("unrecognised address range")
var errBadRange error = errors.New("address range start > end")
var ErrRangeShouldNotBeSpecified error = errors.New("a range may not be specified")

//var addressRangeRE = regexp.MustCompile(`^(?P<address1>` + addressREStr + `)(?P<separator>[,;]?)(?P<address2>` + addressREStr + `).*$`)
var addressRangeRE = regexp.MustCompile(`^(?P<address1>[^,;]*)` + `(?P<separator>[,;]?)` + `(?P<address2>[^,;]*)$`)

/*
An AddressRange stores the start and end addresses of a range.
*/
type AddressRange struct {
	start, end Address
	separator  string // , or ;
}

func (r AddressRange) String() string {
	return fmt.Sprintf("%s%s%s", r.start, r.separator, r.end)
}

/*
 If an address range has been specified, returns the actual start and end line numbers
  as given by calculateStartAndEndLineNumbers.
 Otherwise, returns the current line number as start and end.
*/
func (ra AddressRange) getAddressRange(state *State) (startLine int, endLine int, err error) {
	if !ra.IsAddressRangeSpecified() {
		return state.lineNbr, state.lineNbr, nil
	}
	return ra.calculateStartAndEndLineNumbers(state)
}

/*
 Calculates both start and end line numbers from the given address range.
*/
func (ra AddressRange) calculateStartAndEndLineNumbers(state *State) (startLine int, endLine int, err error) {
	startLine, err = ra.start.calculateActualLineNumber(state)
	if err != nil {
		return 0, 0, err
	}
	endLine, err = ra.end.calculateActualLineNumber(state)
	if err != nil {
		return 0, 0, err
	}
	return startLine, endLine, nil
}

/*
 IsAddressRangeSpecified returns TRUE if the given address range contains valid values.
*/
func (ra AddressRange) IsAddressRangeSpecified() bool {
	return !(ra.start.addr == notSpecified && ra.end.addr == notSpecified)
}

/*
Creates an AddressRange from the given rangeStr string.

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

  Otherwise, a range in format A1,A2 is expected.
 *
 * TODO: A1;A2 not yet supported
*/
func newRange(rangeStr string) (AddressRange, error) {

	// a few special cases to start with
	if rangeStr == "" {
		return AddressRange{Address{addr: notSpecified}, Address{addr: notSpecified}, charComma}, nil
	} else if rangeStr == charDot {
		return AddressRange{Address{addr: currentLine}, Address{addr: currentLine}, charComma}, nil
	} else if rangeStr == charDollar {
		return AddressRange{Address{addr: endOfFile}, Address{addr: endOfFile}, charComma}, nil
	} else if rangeStr == charComma {
		return AddressRange{Address{addr: startOfFile}, Address{addr: endOfFile}, charComma}, nil
	} /* else if justNumberRE.MatchString(rangeStr) {
		// if we can convert to an int, then a simple address has been specified
		// Reason for the RE check above: "+1n" is also convertible to an int, but this has a special meaning
		addrInt, err := strconv.Atoi(rangeStr)
		if err != nil {
			// ignore error, carry on
		} else {
			return AddressRange{Address{addr: addrInt}, Address{addr: addrInt}}, nil
		}
	}

	*/

	var addrRange AddressRange

	matches := findNamedMatches(addressRangeRE, rangeStr)
	if matches == nil {
		return addrRange, errUnrecognisedRange
	}

	start, err := newAddress(matches[firstAddressCaptureGroup])
	if err != nil {
		return addrRange, err
	}
	end, err := newAddress(matches[secondAddressCaptureGroup])
	if err != nil {
		return addrRange, err
	}
	separator := matches[separatorCaptureGroup]

	// special cases: first address empty -> {1,addr} or {.;addr}
	// TODO check if 2nd addr is present
	if start.addr == notSpecified {
		switch separator {
		case separatorComma:
			start = Address{addr: startOfFile}
		case separatorSemicolon:
			start = Address{addr: currentLine}
		}
	}

	// first address given, second empty -> {<given address>, <given address>}
	if end.addr == notSpecified {
		end = start
	}

	/*
		// start must be before end ('special' values excluded)
		if start.addr >= 0 && end.addr >= 0 && start.addr > end.addr {
			return addrRange, errBadRange
		}
	*/
	return AddressRange{start, end, separator}, nil
}
