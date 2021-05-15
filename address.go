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

var errInvalidDestinationAddress error = errors.New("invalid line for destination")
var errUnrecognisedAddress error = errors.New("unrecognised address")

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
