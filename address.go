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
	identDot           string = "."
	identDollar        string = "$"
	identMark          string = "'"
	identRegexForward  string = "/"
	identRegexBackward string = "?"
	identInc           string = "+"
	identDec           string = "-"
	identSignedNbr     string = "1" // this value is only a placeholder, is not parsed as such from the input, nor used in String()
)

// special values for an address
const (
	_            = iota // unused
	currentLine  = -iota
	endOfFile    = -iota
	startOfFile  = -iota
	notSpecified = -iota // if an address is not specified ...
)

/*
addressPart is the parsed version of one capture group in the address-RE.
*/
type addressPart struct {
	addrIdent string // see constant strings ident*
	info      string // set for e.g. mark, regex, signednum
}

/*
Address stores a line number with optional offset
*/
type Address struct {
	addr        int
	offset      int           // only set for +n, -n etc
	specialInfo string        // only set for certain types of addresses
	internal    []addressPart // stores the address as parsed
}

var errInvalidDestinationAddress error = errors.New("invalid line for destination")
var errUnrecognisedAddress error = errors.New("unrecognised address")

/*
Regex for the parts of an address.
 1. special:	. $
 2: mark:		' followed by a lowercase letter
 3: reFor:		/regex/
 4: reBack:		?regex?
 5. signednbr:	+-<n>
 6. incdec:		+ -

These can be repeated any number of times.
Note: the check for a signed number must come before the check for +/-.
*/
var addressRE = regexp.MustCompile(`(?P<dot>\.)|(?P<dollar>\$)|(?P<mark>'[a-z])|(?P<reFor>\/[^/]*\/)|` +
	`(?P<reBack>\?[^\?]*\?)|(?P<signednbr>[+-]?\d+)|(?P<inc>\+)|(?P<dec>-)`)

/*
 Creates a new Address.

 A match is then parsed again using regexp.FindStringSubmatch to identify the capture group.
 (Using this method alone would not allow precise error messages, if at all, in the case of bad input)
*/
func newAddress(addrStr string) (Address, error) {
	addrStr = strings.TrimSpace(addrStr)
	if len(addrStr) == 0 {
		return Address{addr: notSpecified}, nil
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
			// this is ok if reachedhave  end of input; otherwise check the intervening chars
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
				return address, errUnrecognisedAddress
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
 Returns an actual line number, depending on the given address and the current line number if required
*/
func (address Address) calculateActualLineNumber(state *State) (int, error) {
	return address.calculateActualLineNumber2(state.lineNbr, state.Buffer)
}

/*
 calculateActuaLineNumber2 returns an actual line number, depending on the current linenbr and the list of lines.
*/
func (addr Address) calculateActualLineNumber2(currentLineNbr int, buffer *list.List) (int, error) {
	var lineNbr int = currentLineNbr
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
			// TODO
			parsingAddressOffset = true
		case identRegexForward:
			// TODO
			parsingAddressOffset = true
		case identRegexBackward:
			// TODO
			parsingAddressOffset = true
		case identSignedNbr:
			parsedLineNbr, err := strconv.Atoi(addrPart.info)
			if err != nil {
				return -1, fmt.Errorf("error parsing signednbr in address part '%v': %w", addrPart, err)
			}
			if parsingAddressOffset {
				// always relative number in address offset
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
	if lineNbr < 0 || lineNbr > buffer.Len() {
		return -1, errInvalidLine
	}
	return lineNbr, nil
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
/*
TODO
func (a Address) String() string {
	var str string
	if a.special.addrType != 0 {
		var specialStr string
		switch a.special.addrType {
		case mark:
			specialStr = identMark
		case regexBackward:
			specialStr = identRegexBackward
		case regexForward:
			specialStr = identRegexForward
		}
		str = fmt.Sprintf("%s%s", specialStr, a.special.info)
	} else {
		switch a.addr {
		case currentLine:
			str = identDot
		case endOfFile:
			str = identDollar
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
*/

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
