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
		t.Run(fmt.Sprintf("_%s_", test.addressStr), func(t *testing.T) {
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

// these ranges should throw errors (e.g. 'badRange')
func TestRangeErrors(t *testing.T) {
	data := []struct {
		addrRange string
	}{
		{"2,1"},
	}
	for _, test := range data {
		t.Run(fmt.Sprintf("_%s_", test.addrRange), func(t *testing.T) {
			addr, err := newRange(test.addrRange)
			if err != nil {
				// ok
			} else {
				t.Errorf("expected error for input string, got: %v", addr)
			}
		})
	}
}

// tests for ranges with the +/- syntax (where the offset has to be checked)
func TestCreateRangeOffsets(t *testing.T) {
	data := []struct {
		addrRange                          string
		expectedStart, expectedStartOffset int
		expectedEnd, expectedEndOffset     int
	}{
		{"+,5", currentLine, 1, 5, 0},
		{"-,+", currentLine, -1, currentLine, +1},
		{"++,$", currentLine, 2, endOfFile, 0},
		{"+ ++ +,$", currentLine, 4, endOfFile, 0},
		{"---,9", currentLine, -3, 9, 0},
		{"- --  -- ,9", currentLine, -5, 9, 0},
		{"+3,-22", currentLine, 3, currentLine, -22},
		{"-1,+2", currentLine, -1, currentLine, +2},
		{"+1", currentLine, +1, currentLine, +1},
		{"+ 3 , -22", currentLine, 4, currentLine, -22},
		{"+ 3 , - 22", currentLine, 4, currentLine, 21},
	}

	for _, test := range data {
		t.Run(fmt.Sprintf(">>%s<<", test.addrRange), func(t *testing.T) {
			r, err := newRange(test.addrRange)
			if err != nil {
				t.Errorf("error: %s", err)
			} else if r.start.addr != test.expectedStart {
				t.Errorf("bad start: %d, expected: %d", r.start.addr, test.expectedStart)
			} else if r.start.offset != test.expectedStartOffset {
				t.Errorf("bad start offset: %d, expected: %d", r.start.offset, test.expectedStartOffset)
			} else if r.end.addr != test.expectedEnd {
				t.Errorf("bad end: %d, expected: %d", r.end.addr, test.expectedEnd)
			} else if r.end.offset != test.expectedEndOffset {
				t.Errorf("bad end offset: %d, expected: %d", r.end.offset, test.expectedEndOffset)
			}
		})
	}
}

// tests for address ranges, not including +/- syntax
func TestCreateRange(t *testing.T) {
	data := []struct {
		addrRange                  string
		expectedStart, expectedEnd int
	}{
		{"", notSpecified, notSpecified},
		{"1,2", 1, 2},
		{"99,999", 99, 999},
		{"9,", 9, 9},
		{"9", 9, 9},
		{",12", startOfFile, 12},
		{",", startOfFile, endOfFile},
		{"4,$", 4, endOfFile},
		{"$,$", endOfFile, endOfFile},
		{"$", endOfFile, endOfFile},
		{"5", 5, 5},
		{".", currentLine, currentLine},
	}

	for _, test := range data {
		r, err := newRange(test.addrRange)
		if err != nil {
			t.Errorf("error: %s (input string: '%s')", err, test.addrRange)
		} else if r.start.addr != test.expectedStart {
			t.Errorf("bad start: %d, expected: %d (input string: '%s')", r.start.addr, test.expectedStart, test.addrRange)
		} else if r.end.addr != test.expectedEnd {
			t.Errorf("bad end: %d, expected: %d (input string: %s)", r.end.addr, test.expectedEnd, test.addrRange)
		}
	}
}
