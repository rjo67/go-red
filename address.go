package main

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

var invalidLine error = errors.New("invalid line in address range")
var unrecognisedRange error = errors.New("unrecognised address range")
var unrecognisedAddress error = errors.New("unrecognised address")
var badRange error = errors.New("address range start > end")
var rangeShouldNotBeSpecified error = errors.New("a range may not be specified")

type Address struct {
	addr   int
	offset int // only set for +n, -n etc
}

// matches any number of +
var /* const */ allPlusses = regexp.MustCompile("^\\s*[\\+ ]+$")

// matches any number of -
var /* const */ allMinusses = regexp.MustCompile("^\\s*[- ]+$")

// matches +n or -n
var /* const */ plusMinusN = regexp.MustCompile("^\\s*([\\+-])\\s*(\\d+)\\s*$")

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
				return Address{}, unrecognisedAddress
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
			return Address{}, unrecognisedAddress
		}
		nbrStr := matches[0][2]
		var nbr int
		var err error
		if nbrStr == "" { // if empty, throw error
			return Address{}, unrecognisedAddress
		} else {
			nbr, err = strconv.Atoi(nbrStr)
			if err != nil {
				return Address{}, err
			}
		}
		return Address{addr: currentLine, offset: nbr * sign}, nil
	}
	return Address{}, unrecognisedAddress
}

/*
 * Calculates both start and end line numbers from the given address range.
 */
func calculateStartAndEndLineNumbers(addrRange AddressRange, state *State) (startLine int, endLine int, err error) {
	startLine, err = calculateActualLineNumber(addrRange.start, state)
	if err != nil {
		return 0, 0, err
	}
	endLine, err = calculateActualLineNumber(addrRange.end, state)
	if err != nil {
		return 0, 0, err
	}
	return startLine, endLine, nil
}

/*
 * returns an actual line number, depending on the given address
 * and the current line number if required
 */
func calculateActualLineNumber(address Address, state *State) (int, error) {
	var lineNbr int = -99
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
			return -1, invalidLine
		}
		lineNbr = state.buffer.Len() + address.offset
	}
	if lineNbr > state.buffer.Len() {
		return -1, invalidLine
	}
	if lineNbr < 0 || lineNbr > state.buffer.Len() {
		return -1, invalidLine
	} else {
		return lineNbr, nil
	}
}

type AddressRange struct {
	start, end Address
}

/*
 This function returns TRUE if the given address range contains valid values.
 */
func (ra *AddressRange) isAddressRangeSpecified() (bool) {
	return ! (ra.start.addr == notSpecified && ra.end.addr == notSpecified)
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
	} else {
		// if we can convert to an int, then a simple address has been specified
		addrInt, err := strconv.Atoi(rangeStr)
		if err != nil {
			// ignore error, carry on
		} else {
			return AddressRange{Address{addr: addrInt}, Address{addr: addrInt}}, nil
		}
	}

	// here we just split on the , or ; and let newAddress do the hard work
	const addrRE string = "(.*)"
	addrRangeRE := regexp.MustCompile("^" + addrRE + "," + addrRE + "$")
	matches := addrRangeRE.FindAllStringSubmatch(rangeStr, -1)
	// we expect two matches
	if len(matches) != 1 || len(matches[0]) != 3 {
		return addrRange, unrecognisedRange
	}

	startRange := matches[0][1]
	endRange := matches[0][2]
	//fmt.Printf("start '%s', end '%s'\n", startRange, endRange)

	start, err := newAddress(startRange)
	if err != nil {
		return addrRange, err
	}
	end, err := newAddress(endRange)
	if err != nil {
		return addrRange, err
	}

	//------------------------------

	// special cases: first address empty, second given -> {1,addr} or {.;addr}
	if startRange == "" && endRange != "" {
		start = Address{addr: startOfFile}
		//TODO: when using ; set to currentline
	}
	// first address given, second empty -> {1, .}
	if startRange != "" && endRange == "" {
		end = Address{addr: currentLine}
	}

	// start must be before end ('special' values excluded)
	if start.addr >= 0 && end.addr >= 0 && start.addr > end.addr {
		return addrRange, badRange
	}
	return AddressRange{start, end}, nil
}
