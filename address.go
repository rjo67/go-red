package red

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const charComma string = ","
const charDot string = "."
const charDollar string = "$"

// special values for an address
const currentLine int = -1
const endOfFile int = -2
const startOfFile int = -3

// if an address is not specified ...
const notSpecified int = -4

var _ = fmt.Printf // For debugging; delete when done.

var errInvalidLine error = errors.New("invalid line in address range")
var errInvalidDestinationAddress error = errors.New("invalid line for destination")
var errUnrecognisedRange error = errors.New("unrecognised address range")
var errUnrecognisedAddress error = errors.New("unrecognised address")
var errBadRange error = errors.New("address range start > end")
var ErrRangeShouldNotBeSpecified error = errors.New("a range may not be specified")

/*
An Address stores a line number with optional offset
*/
type Address struct {
	addr   int
	offset int // only set for +n, -n etc
}

// matches any number of +
var /* const */ allPlusses = regexp.MustCompile(`^\s*[+ ]+$`)

// matches any number of -
var /* const */ allMinusses = regexp.MustCompile(`^\s*[- ]+$`)

// matches +n or -n
var /* const */ plusMinusN = regexp.MustCompile(`^\s*([+-])\s*(\d+)\s*$`)

/*
 Creates a new Address.

 Special chars:
   An empty string  - notSpecified (let caller decide)
   .                - currentLine
   $                - last line
   +n
    -n
    +
    -

 TODO:
    marks
    regexps
*/
func newAddress(addrStr string) (Address, error) {
	// handle special cases first
	switch addrStr {
	case "":
		return Address{addr: notSpecified}, nil
	case charDot:
		return Address{addr: currentLine}, nil
	case charDollar:
		return Address{addr: endOfFile}, nil
	default:
		matched, trimmedStr := checkRegexAndTrim(allPlusses, addrStr)
		if matched {
			return Address{addr: currentLine, offset: len(trimmedStr)}, nil
		}
		matched, trimmedStr = checkRegexAndTrim(allMinusses, addrStr)
		if matched {
			return Address{addr: currentLine, offset: len(trimmedStr) * -1}, nil
		}
		// try to match +n, -n
		address, err := handlePlusMinusNumber(addrStr)
		//fmt.Printf("1 address %v, err %s\n", address, err)
		if err != nil {
			// last try: just a number
			addrInt, err := strconv.Atoi(addrStr)
			if err != nil {
				return Address{}, errUnrecognisedAddress
			} else {
				return Address{addr: addrInt}, nil
			}
		}
		return address, nil
	}
}

/*
 * if the input matches the regex, TRUE is returned togethe with a copy of input, with all spaces removed.
 */
func checkRegexAndTrim(regex *regexp.Regexp, input string) (bool, string) {
	if regex.MatchString(input) {
		trimmedStr := strings.Replace(input, " ", "", -1)
		return true, trimmedStr
	}
	return false, ""
}

// creates an address from an input +n, -n
// returns an error if wasn't parseable
func handlePlusMinusNumber(addrStr string) (Address, error) {
	matches := plusMinusN.FindAllStringSubmatch(addrStr, -1)
	// we expect two matches
	if len(matches) == 1 && len(matches[0]) == 3 {
		signStr := matches[0][1] // + or -
		var sign int
		switch signStr {
		case "-":
			sign = -1
		case "+":
			sign = 1
		default:
			return Address{}, errUnrecognisedAddress
		}
		nbrStr := matches[0][2]
		var nbr int
		var err error
		if nbrStr == "" { // if empty, throw error
			return Address{}, errUnrecognisedAddress
		} else {
			nbr, err = strconv.Atoi(nbrStr)
			if err != nil {
				return Address{}, err
			}
		}
		return Address{addr: currentLine, offset: nbr * sign}, nil
	}
	return Address{}, errUnrecognisedAddress
}

/*
 Returns an actual line number, depending on the given address and the current line number if required
*/
func (address Address) calculateActualLineNumber(state *State) (int, error) {
	var lineNbr int = -99
	//fmt.Printf("addr: %v\n", address)
	switch {
	case address.addr >= 0:
		// an actual line number has been specified
		lineNbr = address.addr
	case address.addr == currentLine:
		lineNbr = state.lineNbr + address.offset
	case address.addr == startOfFile:
		lineNbr = 1 + address.offset
	case address.addr == endOfFile:
		if address.offset > 0 {
			return -1, errInvalidLine
		}
		lineNbr = state.Buffer.Len() + address.offset
	}
	if lineNbr > state.Buffer.Len() {
		return -1, errInvalidLine
	}
	if lineNbr < 0 || lineNbr > state.Buffer.Len() {
		return -1, errInvalidLine
	} else {
		return lineNbr, nil
	}
}

/*
An AddressRange stores the start and end addresses of a range.
*/
type AddressRange struct {
	start, end Address
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
 * Creates an AddressRange from the given rangeStr string.
 * Special cases:
 *   empty string        {notSpecified, notSpecified}
 *   .                   {currentLine, currentLine}
 *   $                   {endOfFile, endOfFile}
 *   ,                   {startOfFile, endOfFile}
 *   n                   {n, n}
 *
 * Otherwise, a range in format A1,A2 is expected.
 *
 * TODO: A1;A2 not yet supported
 */
func newRange(rangeStr string) (addrRange AddressRange, err error) {
	// a few special cases to start with
	if rangeStr == "" {
		return AddressRange{Address{addr: notSpecified}, Address{addr: notSpecified}}, nil
	} else if rangeStr == charDot {
		return AddressRange{Address{addr: currentLine}, Address{addr: currentLine}}, nil
	} else if rangeStr == charDollar {
		return AddressRange{Address{addr: endOfFile}, Address{addr: endOfFile}}, nil
	} else if rangeStr == charComma {
		return AddressRange{Address{addr: startOfFile}, Address{addr: endOfFile}}, nil
	} else if justNumberRE.MatchString(rangeStr) {
		// if we can convert to an int, then a simple address has been specified
		// Reason for the RE check above: "+1n" is also convertible to an int, but this has a special meaning
		addrInt, err := strconv.Atoi(rangeStr)
		if err != nil {
			// ignore error, carry on
		} else {
			return AddressRange{Address{addr: addrInt}, Address{addr: addrInt}}, nil
		}
	}

	// check here if we've got a comma (or semicolon) - if not, just got one address
	var start, end Address
	if !strings.ContainsAny(rangeStr, ",;") {
		start, err = newAddress(rangeStr)
		if err != nil {
			return addrRange, err
		}
		end = start
	} else {
		// here we just split on the , or ; and let newAddress do the hard work
		const addrRE string = "(.*)"
		addrRangeRE := regexp.MustCompile("^" + addrRE + "," + addrRE + "$")
		matches := addrRangeRE.FindAllStringSubmatch(rangeStr, -1)
		// we expect two matches
		if len(matches) != 1 || len(matches[0]) != 3 {
			return addrRange, errUnrecognisedRange
		}

		startRange := matches[0][1]
		endRange := matches[0][2]

		if start, err = newAddress(startRange); err != nil {
			return addrRange, err
		}
		if end, err = newAddress(endRange); err != nil {
			return addrRange, err
		}

		//------------------------------

		// special cases: first address empty, second given -> {1,addr} or {.;addr}
		if startRange == "" && endRange != "" {
			start = Address{addr: startOfFile}
			//TODO: when using ; set to currentline
		}
		// first address given, second empty -> {<given address>, <given address>}
		if startRange != "" && endRange == "" {
			end = start
		}
	}

	// start must be before end ('special' values excluded)
	if start.addr >= 0 && end.addr >= 0 && start.addr > end.addr {
		return addrRange, errBadRange
	}
	return AddressRange{start, end}, nil
}
