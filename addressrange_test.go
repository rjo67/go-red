package red

import (
	"fmt"
	"testing"
)

// these ranges should throw errors (e.g. 'badRange')
func TestRangeErrors(t *testing.T) {
	data := []struct {
		addrRange string
	}{
		{"2,1"},
		{"1,2--"},
	}
	for _, test := range data {
		t.Run(fmt.Sprintf(">>%s<<", test.addrRange), func(t *testing.T) {
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
		{"1,2", 1, 2},
		{"99,999", 99, 999},
		{"9,", 9, 9},
		{"9", 9, 9},
		{",12", startOfFile, 12}, // If only the second address is given, the resulting address pairs are '1,addr' and '.;addr' respectively
		{";12", currentLine, 12}, // If only the second address is given, the resulting address pairs are '1,addr' and '.;addr' respectively
		{",", startOfFile, endOfFile},
		{"4,$", 4, endOfFile},
		{"$,$", endOfFile, endOfFile},
		{"$", endOfFile, endOfFile},
		{"5", 5, 5},
		{"", notSpecified, notSpecified},
		{".", currentLine, currentLine},
	}

	for _, test := range data {
		t.Run(fmt.Sprintf(">>%s<<", test.addrRange), func(t *testing.T) {
			r, err := newRange(test.addrRange)
			if err != nil {
				t.Errorf("error: %s", err)
			} else if r.start.addr != test.expectedStart {
				t.Errorf("bad start: %d, expected: %d", r.start.addr, test.expectedStart)
			} else if r.end.addr != test.expectedEnd {
				t.Errorf("bad end: %d, expected: %d", r.end.addr, test.expectedEnd)
			}
		})
	}
}

func TestSpecialRange(t *testing.T) {
	data := []struct {
		addrRange         string
		expectedStartType int
		expectedStartInfo string
		expectedEndType   int
		expectedEndInfo   string
	}{
		{"'a,'b", mark, "a", mark, "b"},
		{"'a,/123/", mark, "a", regexForward, "123"},
		{"/123/,/456/", regexForward, "123", regexForward, "456"},
		{"/123/,?456?", regexForward, "123", regexBackward, "456"},
		{"'b,?abc?", mark, "b", regexBackward, "abc"},
		{"?abc?,'d", regexBackward, "abc", mark, "d"},
		// some examples with spaces
		{"'a ,'b", mark, "a", mark, "b"},
		{"'a, /123/", mark, "a", regexForward, "123"},
		{"/123/ ,/456/    ", regexForward, "123", regexForward, "456"},
		{"   /123/,?456?", regexForward, "123", regexBackward, "456"},
		{"'b  ,  ?abc?", mark, "b", regexBackward, "abc"},
		{"?abc?,  'd", regexBackward, "abc", mark, "d"},
	}

	for _, test := range data {
		t.Run(fmt.Sprintf(">>%s<<", test.addrRange), func(t *testing.T) {
			r, err := newRange(test.addrRange)
			if err != nil {
				t.Errorf("error: %s", err)
			} else if r.start.special.addrType != test.expectedStartType {
				t.Errorf("bad start type: %d, expected: %d", r.start.special.addrType, test.expectedStartType)
			} else if r.start.special.info != test.expectedStartInfo {
				t.Errorf("bad start info: %s, expected: %s", r.start.special.info, test.expectedStartInfo)
			} else if r.end.special.addrType != test.expectedEndType {
				t.Errorf("bad end type: %d, expected: %d", r.end.special.addrType, test.expectedEndType)
			} else if r.end.special.info != test.expectedEndInfo {
				t.Errorf("bad end info: %s, expected: %s", r.end.special.info, test.expectedEndInfo)
			}
		})
	}
}
