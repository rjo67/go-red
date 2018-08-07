package main

import (
	"fmt"
	"regexp"
	"strconv"
)

const charComma string = ","
const charDot string = "."
const charDollar string = "$"

// special values for an address
const currentLine int = -1
const endOfFile int = -2
const startOfFile int = -3

var _ = fmt.Printf // For debugging; delete when done.

type AddressError struct {
	desc string
}

func (e *AddressError) Error() string {
	return e.desc
}

var invalidLine AddressError = AddressError{"invalid line in address range"}
var unrecognisedRange AddressError = AddressError{"unrecognised address range"}
var unrecognisedAddress AddressError = AddressError{"unrecognised address"}
var badRange AddressError = AddressError{"address range start > end"}

type Address struct {
	addr   int
	offset int // only set for +n, -n etc
}

/*
 * Creates a new Address.
 * Special chars:
 *  An empty string  - currentLine
 *  .  currentLine
 *  $  last line
 *  +n
 *  -n
 *  +
 *  -
 *
 * TODO:
 *  marks
 *  regexps
 */
func newAddress(addrStr string) (Address, error) {
	switch addrStr {
	case "":
		return Address{addr: currentLine}, nil
	case charDot:
		return Address{addr: currentLine}, nil
	case charDollar:
		return Address{addr: endOfFile}, nil
	default:
		// try to match +n, -n
		var address Address
		var err error
		address, err = handlePlusMinusNumber(addrStr)
		//fmt.Printf("1 address %v, err %s\n", address, err)
		if err != nil {
			// try to match any number of +, -
			address, err = handlePlusMinus(addrStr)
			//fmt.Printf("2 address %v, err %s\n", address, err)
			if err != nil {
				// last try: just a number
				addrInt, err := strconv.Atoi(addrStr)
				if err != nil {
					return Address{}, err
				} else {
					return Address{addr: addrInt}, nil
				}
			}
		}
		return address, nil
	}
}

// creates an address from an input of any number of + or -, e.g. +++, --
// returns an error if wasn't parseable
func handlePlusMinus(addrStr string) (Address, error) {
	signRE := regexp.MustCompile("(^-*$)|(^\\+*$)")
	matches := signRE.FindAllStringSubmatch(addrStr, -1)
	// either doesn't match, or matches any number of "-" in first match
	// or any number of "+" in second match
	// we expect two matches
	if len(matches) == 1 && len(matches[0]) == 3 {
		var match string
		var sign int
		if matches[0][1] != "" {
			// matches on minus
			sign = -1
			match = matches[0][1]
		} else {
			sign = 1
			match = matches[0][2]
		}
		return Address{addr: currentLine, offset: len(match) * sign}, nil
	}
	return Address{}, &unrecognisedAddress
}

// creates an address from an input +n, -n
// returns an error if wasn't parseable
func handlePlusMinusNumber(addrStr string) (Address, error) {
	signRE := regexp.MustCompile("^(\\+|-)(\\d+)")
	matches := signRE.FindAllStringSubmatch(addrStr, -1)
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
			return Address{}, &unrecognisedAddress
		}
		nbrStr := matches[0][2]
		var nbr int
		var err error
		if nbrStr == "" { // if empty, throw error and try the next pattern
			return Address{}, &unrecognisedAddress
		} else {
			nbr, err = strconv.Atoi(nbrStr)
			if err != nil {
				return Address{}, err
			}
		}
		return Address{addr: currentLine, offset: nbr * sign}, nil
	}
	return Address{}, &unrecognisedAddress
}

type AddressRange struct {
	start, end Address
}

/*
 * Creates an AddressRange from the given rangeStr string.
 * An empty string implies a range of {currentLine, currentLine}
 */
func newRange(rangeStr string) (addrRange AddressRange, err error) {
	// a few special cases to start with
	if rangeStr == "" || rangeStr == charDot {
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

	// an address is a number or special chars $ . + -
	// this allows +++5 which is invalid (will be caught later)
	const addrRE string = "\\s*(\\d+|\\$|\\.|\\++\\d*|-+\\d*)?"
	addrRangeRE := regexp.MustCompile("^" + addrRE + "," + addrRE + "$")
	matches := addrRangeRE.FindAllStringSubmatch(rangeStr, -1)
	// we expect two matches
	if len(matches) != 1 || len(matches[0]) != 3 {
		return addrRange, &unrecognisedRange
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

	// special cases: start empty, end not empty, or v.v.
	if startRange == "" && endRange != "" {
		start = Address{addr: startOfFile}
	}
	if startRange != "" && endRange == "" {
		end = Address{addr: currentLine}
	}

	// start must be before end ('special' values excluded)
	if start.addr >= 0 && end.addr >= 0 && start.addr > end.addr {
		return addrRange, &badRange
	}
	return AddressRange{start, end}, nil
}
