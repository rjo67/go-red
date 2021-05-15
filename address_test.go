package red

import (
	"fmt"
	"testing"
)

func TestParseAddressSpecials(t *testing.T) {
	data := []struct {
		addressStr   string
		expectedType int
		expectedInfo string
	}{
		// marks
		{"'a", mark, "a"},
		{"/.*/", regexForward, ".*"},
		{"?.*?", regexBackward, ".*"},
	}
	for _, test := range data {
		t.Run(fmt.Sprintf("_%s_", test.addressStr), func(t *testing.T) {
			addr, err := newAddress(test.addressStr)
			if err != nil {
				t.Errorf("error: %s", err)
			} else if addr.special.addrType != test.expectedType {
				t.Errorf("bad type, got: %d, expected: %d", addr.special.addrType, test.expectedType)
			} else if addr.special.info != test.expectedInfo {
				t.Errorf("bad info, got: %s, expected: %s", addr.special.info, test.expectedInfo)
			} else if addr.addr != 0 || addr.offset != 0 {
				t.Errorf("addr/offset must be zero for 'special' addresses, got addr: %d, offset: %d", addr.addr, addr.offset)
			}
		})
	}
}

func TestParseAddress(t *testing.T) {
	data := []struct {
		addressStr                         string
		expectedStart, expectedStartOffset int
	}{
		{"", notSpecified, 0},
		{"+", currentLine, 1},
		{"++", currentLine, 2},
		{"+++++", currentLine, 5},
		{"-", currentLine, -1},
		{"---", currentLine, -3},
		{"-----", currentLine, -5},
		{"++---", currentLine, -1},
		{"+++--", currentLine, 1},
		{"+++--+++", currentLine, 4},
		{"--++", currentLine, 0},
		{"3", 3, 0},
		{"++-+3", currentLine, 4},
		{"21", 21, 0},
		{"+2", currentLine, 2},
		{"++2", currentLine, 3},
		{"+23", currentLine, 23},
		{"----------+23", currentLine, 13},
		{"-3", currentLine, -3},
		{"+-3", currentLine, -2},
		{"-31", currentLine, -31},
		// whitespace
		{"+ +", currentLine, 2},
		{"+ ++ ++", currentLine, 5},
		{"- -", currentLine, -2},
		{"- -- --", currentLine, -5},
		{"+ +2", currentLine, 3},
		{"+ -2", currentLine, -1},
		{"+ - ++ 2", currentLine, 4},
		// . $
		{"$", endOfFile, 0},
		{"$-1", endOfFile, -1},
		{"$--3", endOfFile, -4},
		{".", currentLine, 0},
		{".+1", currentLine, 1},
		{".1", currentLine, 1},
		{".++1", currentLine, 2},
		{".++4", currentLine, 5},
		{"$1", endOfFile, 1}, // syntactically legal, although an invalid range
		{".1", currentLine, 1},
	}

	for _, test := range data {
		t.Run(fmt.Sprintf(">>%s<<", test.addressStr), func(t *testing.T) {
			addr, err := newAddress(test.addressStr)
			if err != nil {
				t.Errorf("error: %s", err)
			} else if addr.addr != test.expectedStart {
				t.Errorf("bad start, got: %d, expected: %d", addr.addr, test.expectedStart)
			} else if addr.offset != test.expectedStartOffset {
				t.Errorf("bad start offset, got: %d, expected: %d", addr.offset, test.expectedStartOffset)
			}
		})
	}
}

func TestAddressErrors(t *testing.T) {
	data := []struct {
		addressStr string
	}{
		{"'qwe"}, // mark can only be one letter
		{"/we"},  // non-terminated regex
	}
	for _, test := range data {
		t.Run(fmt.Sprintf("_%s_", test.addressStr), func(t *testing.T) {
			addr, err := newAddress(test.addressStr)
			if err != nil {
				// ok
			} else {
				t.Errorf("expected error for input string, got: %v", addr)
			}
		})
	}
}
