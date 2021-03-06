package red

import (
	"container/list"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// identifiers, used e.g. in addressPart
const (
	identComma         string = ","
	identDec           string = "-"
	identDot           string = "."
	identDollar        string = "$"
	identInc           string = "+"
	identMark          string = "'"
	identNotSpecified  string = "XXX" // if an address is not specified ...
	identRegexBackward string = "?"
	identRegexForward  string = "/"
	identSemicolon     string = ";"
	identSignedNbr     string = "1" // this value is only a placeholder, is not parsed as such from the input, nor used in String()
)

// special values for an address (part)
const (
	_           = iota // unused
	currentLine = -iota
	endOfFile   = -iota
	startOfFile = -iota
)

/*
addressPart is the parsed version of one capture group in the address-RE.
*/
type addressPart struct {
	addrIdent string // see constant strings ident*
	info      string // set for e.g. mark, regex, signednum
}

/*
Address stores a list of AddressParts, which taken together will result in an actual line number.

The AddressPart representation is independent of the current 'state'.
Use calculateActuaLineNumber2(...) to convert the internal representation into an actual line number.
*/
type Address struct {
	internal []addressPart // stores the address as parsed
}

// AddressError records an address error.
type AddressError struct {
	Msg string
	Err error
}

func (e *AddressError) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	} else {
		return e.Msg
	}
}

func (e *AddressError) Unwrap() error { return e.Err }

func errorInvalidLine(str string, err error) error {
	return &AddressError{Msg: fmt.Sprintf("invalid line: %s", str), Err: err}
}

// errorInvalidDestination is used by commands which take a 'destination'
func errorInvalidDestination(str string, err error) error {
	return &AddressError{Msg: fmt.Sprintf("invalid destination: %s", str), Err: err}
}
func errorUnrecognisedAddress(str string, err error) error {
	return &AddressError{Msg: fmt.Sprintf("unrecognised address: %s", str), Err: err}
}

/*
Regex for the parts of an address.
 1. special:	. $
 2: mark:		' followed by a lowercase letter
 3: reFor:		/regex/
 4: reBack:		?regex?
 5. signednbr:	+-<n>
 6. inc:		+
 7. dec:		-

These can be repeated any number of times.
Note: the check for a signed number must come before the check for +/-.
*/
var addressRE = regexp.MustCompile(`(?P<dot>\.)|(?P<dollar>\$)|(?P<mark>'[a-z])|(?P<reFor>\/[^/]*\/)|` +
	`(?P<reBack>\?[^\?]*\?)|(?P<signednbr>[+-]?\d+)|(?P<inc>\+)|(?P<dec>-)`)

/*
isNotSpecified returns true if this address was not specified.
*/
func (a Address) isNotSpecified() bool {
	return a.internal[0].addrIdent == identNotSpecified
}

/*
isSpecified returns true if this address was specified.
*/
func (a Address) isSpecified() bool {
	return !a.isNotSpecified()
}

/*
newUnspecifiedAddress creates a new Address object with a special AddressPart to indiacte 'not specified'.
*/
func newUnspecifiedAddress() Address {
	parts := make([]addressPart, 1)
	parts[0] = addressPart{addrIdent: identNotSpecified}
	return Address{internal: parts}
}

/*
newAbsoluteAddress creates a new Address which references a given absolute line number.
*/
func newAbsoluteAddress(lineNbr int) Address {
	parts := make([]addressPart, 1)
	parts[0] = addressPart{addrIdent: identSignedNbr, info: strconv.Itoa(lineNbr)}
	return Address{internal: parts}
}

/*
 Creates a new Address from an input string.
/*
 Creates a new Address from an input string.
*/
func newAddress(addrStr string) (Address, error) {
	addrStr = strings.TrimSpace(addrStr)
	if len(addrStr) == 0 {
		return newUnspecifiedAddress(), nil
	}

	// stores the parsed address parts
	address := Address{}

	/*
	 Iterate over the input string by using regexp.FindStringIndex.
	 This is to be able to pick up input errors, such as a non-terminated regex, by detecting if any non-blank chars were skipped over.
	*/
	index := 0 // the index of the input string slice where the current RE matched
	keepGoing := true
	for keepGoing {
		loc := addressRE.FindStringIndex(addrStr[index:])
		if loc == nil {
			// this is ok if have reached end of input; otherwise check the intervening chars
			if index != len(addrStr) {
				err := checkSkippedChars(index, len(addrStr), addrStr)
				if err != nil {
					return address, err
				}
			}
			keepGoing = false
			continue
		}
		startOfMatch := index + loc[0]
		if loc[0] != 0 {
			// char(s) were skipped during the previous call of FindStringIndex -- either spaces or error
			err := checkSkippedChars(index, startOfMatch, addrStr)
			if err != nil {
				return address, err
			}
		}

		// have now matched the RE at startOfMatch:index+loc[1]

		//fmt.Printf("%02d-%02d >>%s<<\n", startOfMatch, index+loc[1], addrStr[startOfMatch:index+loc[1]])

		// re-execute RE find to get named capture groups
		matches := findNamedMatches(addressRE, addrStr[startOfMatch:index+loc[1]], false)

		var addrPart addressPart
		switch {
		case len(matches["dot"]) != 0:
			addrPart = addressPart{addrIdent: identDot}
		case len(matches["dollar"]) != 0:
			addrPart = addressPart{addrIdent: identDollar}
		case len(matches["mark"]) != 0:
			addrPart = addressPart{addrIdent: identMark, info: matches["mark"][1:]}
		case len(matches["reFor"]) != 0:
			addrPart = addressPart{addrIdent: identRegexForward, info: matches["reFor"][1 : len(matches["reFor"])-1]}
		case len(matches["reBack"]) != 0:
			addrPart = addressPart{addrIdent: identRegexBackward, info: matches["reBack"][1 : len(matches["reBack"])-1]}
		case len(matches["inc"]) != 0:
			addrPart = addressPart{addrIdent: identInc}
		case len(matches["dec"]) != 0:
			addrPart = addressPart{addrIdent: identDec}
		case len(matches["signednbr"]) != 0:
			// check the number is valid (should be ok, since the RE matched, but still...)
			_, err := strconv.Atoi(matches["signednbr"])
			if err != nil {
				return address, errorUnrecognisedAddress(matches["signednbr"], err)
			}
			addrPart = addressPart{addrIdent: identSignedNbr, info: matches["signednbr"]}
		default:
			return address, errors.New("apparently no RE match second time around ... should not happen")
		}

		address.internal = append(address.internal, addrPart)

		index += loc[1]
	}

	return address, nil
}

/**
addressPartsAsString returns the parsed addressedParts as a comma-separated string.
*/
func (addr Address) addressPartsAsString() string {
	var addressParts string
	for _, part := range addr.internal {
		addressParts += part.String() + ","
	}
	// remove trailing comma
	if len(addressParts) != 0 {
		addressParts = addressParts[:len(addressParts)-1]
	}
	return addressParts
}

/*
 calculateActuaLineNumber calculates and returns the actual line number specified by the address,
 depending on the current linenbr and the list of lines. (The list of marks can also be required.)
 Returns error errInvalidLine if the resulting line number is out-of-bounds (<0, > max).
*/
func (addr Address) calculateActualLineNumber(currentLineNbr int, buffer *list.List, marks map[string]int) (int, error) {
	var lineNbr int = currentLineNbr
	if addr.isNotSpecified() {
		return currentLineNbr, nil
	}
	parsingAddressOffset := false // if true, all numbers (e.g. 2, or +2) are treated as offsets
	//fmt.Printf("addr: %v\n", address)
	for _, addrPart := range addr.internal {
		switch addrPart.addrIdent {
		case identInc:
			lineNbr++
			parsingAddressOffset = true
		case identDec:
			lineNbr--
			parsingAddressOffset = true
		case identDollar:
			lineNbr = buffer.Len()
			parsingAddressOffset = true
		case identDot:
			// noop - ignored
			parsingAddressOffset = true
		case identMark:
			if markLineNbr, markDefined := marks[addrPart.info]; markDefined {
				lineNbr = markLineNbr
				parsingAddressOffset = true
			} else {
				return -1, fmt.Errorf("unknown mark: '%s'", addrPart.info)
			}
		case identRegexForward:
			matchingLineNbr, err := matchLineForward(lineNbr, addrPart.info, buffer)
			if err != nil {
				return -1, fmt.Errorf("did not find line matching regex")
			}
			lineNbr = matchingLineNbr
			parsingAddressOffset = true
		case identRegexBackward:
			matchingLineNbr, err := matchLineBackward(lineNbr, addrPart.info, buffer)
			if err != nil {
				return -1, fmt.Errorf("did not find line matching regex")
			}
			lineNbr = matchingLineNbr
			parsingAddressOffset = true
		case identSignedNbr:
			parsedLineNbr, err := strconv.Atoi(addrPart.info)
			if err != nil {
				return -1, fmt.Errorf("error parsing signednbr in address part '%v': %w", addrPart, err)
			}
			if parsingAddressOffset {
				// always relative number when parsing address offset
				lineNbr += parsedLineNbr
			} else {
				switch addrPart.info[0:1] {
				case "+", "-":
					// relative number
					lineNbr += parsedLineNbr
					parsingAddressOffset = true
				default:
					// absolute nbr
					lineNbr = parsedLineNbr
					parsingAddressOffset = true
				}
			}
		default:
			return -1, fmt.Errorf("address part '%s' not recognised", addrPart.addrIdent)
		}
	}
	if lineNbr < 0 {
		return -1, errorInvalidLine(fmt.Sprintf("%d", lineNbr), nil)
	}
	if lineNbr > buffer.Len() {
		return -1, errorInvalidLine(fmt.Sprintf("%d, max line: %d", lineNbr, buffer.Len()), nil)
	}
	return lineNbr, nil
}

/*
Returns the line number of the next line after 'startLine' which matches the given regex.
Search will wrap around.
*/
func matchLineForward(startLine int, reStr string, buffer *list.List) (int, error) {
	re := regexp.MustCompile(reStr)

	// move to line 'startLine'
	e := _findLine(startLine, buffer)
	// should not happen
	if e == nil {
		return -1, fmt.Errorf("matchLineForward: move to line '%d' failed", startLine)
	}

	found := false
	currentLineNbr := startLine

	// starting at the next line, iterate to end of file matching regex
	for e = e.Next(); e != nil && !found; e = e.Next() {
		currentLineNbr++
		if re.MatchString(e.Value.(Line).Line) {
			found = true
		}
	}

	if found {
		return currentLineNbr, nil
	}

	// now iterate from start of file to 'startLine' matching regex
	for currentLineNbr, e = 0, buffer.Front(); (currentLineNbr != startLine) && !found; e = e.Next() {
		currentLineNbr++
		if re.MatchString(e.Value.(Line).Line) {
			found = true
		}
	}

	if found {
		return currentLineNbr, nil
	} else {
		return -1, fmt.Errorf("mo matching line found")
	}
}

/*
Returns the line number of the first line before 'startLine' which matches the given regex.
Search will wrap around.
*/
func matchLineBackward(startLine int, reStr string, buffer *list.List) (int, error) {
	re := regexp.MustCompile(reStr)

	// move to line 'startLine'
	e := _findLine(startLine, buffer)
	// should not happen
	if e == nil {
		return -1, fmt.Errorf("matchLineForward: move to line '%d' failed", startLine)
	}

	currentLineNbr := startLine

	// starting at the previous line, iterate to start of file matching regex
	for e = e.Prev(); e != nil; e = e.Prev() {
		currentLineNbr--
		if re.MatchString(e.Value.(Line).Line) {
			return currentLineNbr, nil
		}
	}

	// now iterate from end of file back to 'startLine' matching regex
	for currentLineNbr, e = buffer.Len(), buffer.Back(); currentLineNbr != startLine; e, currentLineNbr = e.Prev(), currentLineNbr-1 {
		if re.MatchString(e.Value.(Line).Line) {
			return currentLineNbr, nil
		}
	}

	return -1, fmt.Errorf("mo matching line found")
}

/*
syntaxError generates a new error with the given text.
If errorText is empty, will generate a general error message.
*/
func syntaxError(errorText string) error {
	if len(strings.TrimSpace(errorText)) == 0 {
		return errors.New("syntax error")
	}
	return errors.New("syntax error: " + errorText)
}

/*
checkSkippedChars checks if there were any non-blank chars between 'start' and 'end'. If so, returns an error.
*/
func checkSkippedChars(start, end int, str string) error {
	trimmed := strings.TrimSpace(str[start:end])
	if len(trimmed) != 0 {
		locnString, errorDesc := analyzeParseError(start, end, trimmed)
		return syntaxError(fmt.Sprintf("%s at posn %s: >>%s<<", errorDesc, locnString, str[start:end]))
	}
	return nil
}

func analyzeParseError(start, end int, str string) (string, string) {
	var locnString, errorDesc string
	if start+1 == end {
		locnString = strconv.Itoa(start)
	} else {
		locnString = fmt.Sprintf("%d-%d", start, end)
	}
	switch str[0:1] {
	case identMark:
		errorDesc = "incorrect mark"
	case identRegexForward, identRegexBackward:
		errorDesc = "non-terminated regex"
	default:
		errorDesc = "unrecognised"
	}
	return locnString, errorDesc
}

/*
String generates a pretty form of an Address.
*/
func (a Address) String() string {
	if a.isNotSpecified() {
		return fmt.Sprintf("(%s)", "not specified")
	}
	var str string
	for _, part := range a.internal {
		str = fmt.Sprintf("%s%s,", str, part)
	}
	return fmt.Sprintf("(%s)", str[0:len(str)-1])
}

/*
String generates a pretty form of an addressPart.
*/
func (p addressPart) String() string {
	switch p.addrIdent {
	case identMark:
		return fmt.Sprintf("%s%s", identMark, p.info)
	case identRegexBackward, identRegexForward:
		return fmt.Sprintf("%s%s%s", p.addrIdent, p.info, p.addrIdent)
	case identInc, identDec, identDollar, identDot:
		return p.addrIdent
	case identSignedNbr:
		return p.info
	default:
		return fmt.Sprintf("not recognised: '%s'", p.addrIdent)

		// start of file!?
	}
}
