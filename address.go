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
const (
	_            = iota // unused
	currentLine  = -iota
	endOfFile    = -iota
	startOfFile  = -iota
	notSpecified = -iota // if an address is not specified ...
)

var _ = fmt.Printf // For debugging; delete when done.

var errInvalidLine error = errors.New("invalid line in address range")
var errInvalidDestinationAddress error = errors.New("invalid line for destination")
var errUnrecognisedRange error = errors.New("unrecognised address range")
var errUnrecognisedAddress error = errors.New("unrecognised address")
var errBadRange error = errors.New("address range start > end")
var ErrRangeShouldNotBeSpecified error = errors.New("a range may not be specified")

/*
Regex for an address.
First part:
 1a: any number of + or - (mixed). The last + or - binds to the number, if present. See "second part".
e.g. ++--5  == ++- -5
     ++++   == +++ +   (no number in this case)

 1b: the chars . or $
 1c: char ' followed by a lowercase letter  ( == a mark)
 1d: /regex/ or ?regex?

Second part: optional sign followed by a number

*/
const addressREStr string = `^([\.\$ +-]*?||'[a-z]|/.*/|\?.*\?)([+-]?)(\d*) *$`

var addressRE = regexp.MustCompile(addressREStr)
var addressRangeRE = regexp.MustCompile("^" + addressREStr + "[,;]" + addressREStr + ".*$")

// indicators for certain types of address
const (
	_             = iota
	mark          = iota
	regexForward  = iota
	regexBackward = iota
)

/*
SpecialAddress stores extra information for certain types of addresses (marks, regex).
*/
type specialAddress struct {
	addrType int
	info     string
}

/*
Address stores a line number with optional offset
*/
type Address struct {
	addr    int
	offset  int            // only set for +n, -n etc
	special specialAddress // only set for certain types of addresses
}

func (a Address) String() string {
	var str string
	if a.special.addrType != 0 {
		var specialStr string
		switch a.special.addrType {
		case mark:
			specialStr = "'"
		case regexBackward:
			specialStr = "?"
		case regexForward:
			specialStr = "/"
		}
		str = fmt.Sprintf("%s%s", specialStr, a.special.info)
	} else {
		switch a.addr {
		case currentLine:
			str = "."
		case endOfFile:
			str = "$"
		case startOfFile:
			str = "<startoffile>"
		default:
			str = fmt.Sprintf("%d", a.addr)
		}
		if a.offset != 0 {
			str = fmt.Sprintf("%s,%d", str, a.offset)
		}
	}
	return fmt.Sprintf("(%s)", str)
}

/*
 Creates a new Address.

TODO  ??
   An empty string  - notSpecified (let caller decide)
*/
func newAddress(addrStr string) (Address, error) {
	matches := addressRE.FindStringSubmatch(addrStr)
	if matches == nil {
		return Address{}, errUnrecognisedAddress
	}
	// debugging***************
	/*
		for i, match := range matches[1:] {
			if strings.TrimSpace(match) != "" {
				fmt.Printf("match %d: %v\n", i+1, match)
			}
		}
	*/
	// debugging end***********

	// 'special' chars are in matches[1]
	if len(matches[1]) != 0 {
		switch matches[1][0] {
		case '\'':
			// marks
			return Address{special: specialAddress{addrType: mark, info: matches[1][1:]}}, nil
		case '/':
			// regex forward
			return Address{special: specialAddress{addrType: regexForward, info: matches[1][1 : len(matches[1])-1]}}, nil
		case '?':
			// regex backward
			return Address{special: specialAddress{addrType: regexBackward, info: matches[1][1 : len(matches[1])-1]}}, nil
		}
	}

	// handle chars . $ and any number of +/-
	cnt, foundCurrentLine, foundEndOfFile := 0, false, false
	for _, ch := range strings.TrimSpace(matches[1]) {
		switch ch {
		case '+':
			cnt++
		case '-':
			cnt--
		case '.':
			foundCurrentLine = true
		case '$':
			foundEndOfFile = true
		}
	}

	signStr := strings.TrimSpace(matches[2])
	numStr := strings.TrimSpace(matches[3])
	foundSign := len(signStr) != 0
	foundNumber := len(numStr) != 0
	addrOffset := 0
	// avoid processing if both were empty
	if foundSign || foundNumber {
		// handle case of only +/- without a number by adding '1', to enable the Atoi
		if !foundNumber {
			numStr = "1"
		}
		if foundSign {
			numStr = signStr + numStr
		}
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return Address{}, errUnrecognisedAddress
		}
		addrOffset = num
	}

	if foundCurrentLine {
		return Address{addr: currentLine, offset: cnt + addrOffset}, nil
	} else if foundEndOfFile {
		return Address{addr: endOfFile, offset: cnt + addrOffset}, nil
	} else {
		// an absolute address is present if only a number was specified (e.g. '3').
		// Note: the presence of a sign implies a relative address. -ve numbers are always relative.
		// '2'   absolute
		// '-2'  relative
		// '+2'  relative
		// '++2' relative
		// '++'  relative
		if cnt == 0 && !foundSign && foundNumber {
			return Address{addr: cnt + addrOffset, offset: 0}, nil
		}
		return Address{addr: currentLine, offset: cnt + addrOffset}, nil
	}
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

func (r AddressRange) String() string {
	return fmt.Sprintf("%s,%s", r.start, r.end)
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
