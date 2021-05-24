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
		{"+3,-2"},
		{"+ 3 , -2"},
	}
	for _, test := range data {
		t.Run(fmt.Sprintf(">>%s<<", test.addrRange), func(t *testing.T) {
			r, err := newRange(test.addrRange)
			if err != nil {
				t.Errorf("error: %s", err)
			} else {
				start, end, err := r.calculateStartAndEndLineNumbers(1, createListOfLines([]string{"1", "2", "3", "4 123", "5", "6 456regex", "7", "8"}))
				if err != nil {
					// ok
				} else {
					t.Errorf("expected range error, got start=%d, end=%d", start, end)
				}
			}
		})
	}
}

// tests for ranges, checking the 'real' line numbers
func TestCreateAddressRange(t *testing.T) {
	data := []struct {
		addrRange                  string
		startLine                  int
		expectedStart, expectedEnd int
	}{
		{"+,5", 2, 3, 5},
		{"-,+", 2, 1, 3},
		{"++,$", 1, 3, 8},
		{"+ ++ +,$", 3, 7, 8},
		{"---,8", 8, 5, 8},
		{"- --  -- ,8", 7, 2, 8},
		{"-1,+2", 3, 2, 5},
		{"+1", 4, 5, 5},
		{"+ 2 , +3", 4, 7, 7},
		{"1,2", 1, 1, 2},
		{"7,8", 2, 7, 8},
		{"7,", 5, 7, 7},
		{"8", 3, 8, 8},
		{",7", 2, 1, 7}, // If only the second address is given, the resulting address pair is '1,addr'
		{";8", 2, 2, 8}, // If only the second address is given, the resulting address pair is '.;addr'
		{",", 3, 1, 8},  // (1,$)
		{";", 3, 3, 8},  // (.,$)
		{"4,$", 2, 4, 8},
		{"$,$", 4, 8, 8},
		{"$", 3, 8, 8},
		{"5", 3, 5, 5},
		{".", 5, 5, 5},
		// marks
		//{"'a,'b", mark, "a", mark, "b"},
		//{"'a,/123/", mark, "a", regexForward, "123"},
		//{"'a ,'b", mark, "a", mark, "b"},
		//{"'b,?abc?", mark, "b", regexBackward, "abc"},
		//{"'a, /123/", mark, "a", regexForward, "123"},
		//{"'b  ,  ?abc?", mark, "b", regexBackward, "abc"},
		//{"?abc?,'d", regexBackward, "abc", mark, "d"},
		//{"?abc?,  'd", regexBackward, "abc", mark, "d"},
		// regex
		{"/123/,/456/", 2, 4, 6},
		{"/123/ ,/456/    ", 1, 4, 6},
		{"/123/,?456?", 1, 4, 6},
		{"   /123/,?456?", 1, 4, 6},
	}

	for _, test := range data {
		t.Run(fmt.Sprintf(">>%s<<", test.addrRange), func(t *testing.T) {
			r, err := newRange(test.addrRange)
			if err != nil {
				t.Errorf("error: %s", err)
			} else {
				start, end, err := r.calculateStartAndEndLineNumbers(test.startLine, createListOfLines([]string{"1", "2", "3", "4 123", "5", "6 456regex", "7", "8"}))
				if err != nil {
					t.Errorf("error: %s", err)
				}
				assertInt(t, "bad start", start, test.expectedStart)
				assertInt(t, "bad end", end, test.expectedEnd)
			}
		})
	}
	// test empty input
	r, err := newRange("")
	if err != nil {
		t.Errorf("error: %s", err)
	}
	if r.IsSpecified() {
		t.Errorf("expected not specified, got %s", r)
	}

}
