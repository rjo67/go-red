package main

import (
	"testing"
)

// expects an error 'badRange'
func TestBadRange(t *testing.T) {
	input := "2,1"
	_, err := newRange(input)
	if err != nil {
		if err != &badRange {
			t.Fatalf("error: %s (input string: %s)", err, input)
		}
	} else {
		t.Fatalf("expected error for input string: %s", input)
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
		{"---,9", currentLine, -3, 9, 0},
		{"+3,-22", currentLine, 3, currentLine, -22},
		{"-1,+2", currentLine, -1, currentLine, +2},
	}

	for _, test := range data {
		r, err := newRange(test.addrRange)
		if err != nil {
			t.Errorf("error: %s (input string: %s)", err, test.addrRange)
		} else if r.start.addr != test.expectedStart {
			t.Errorf("bad start: %d, expected: %d (input string: %s)", r.start.addr, test.expectedStart, test.addrRange)
		} else if r.start.offset != test.expectedStartOffset {
			t.Errorf("bad start offset: %d, expected: %d (input string: %s)", r.start.offset, test.expectedStartOffset, test.addrRange)
		} else if r.end.addr != test.expectedEnd {
			t.Errorf("bad end: %d, expected: %d (input string: %s)", r.end.addr, test.expectedEnd, test.addrRange)
		} else if r.end.offset != test.expectedEndOffset {
			t.Errorf("bad end offset: %d, expected: %d (input string: %s)", r.end.offset, test.expectedEndOffset, test.addrRange)
		}
	}
}

// tests for address ranges, not including +/- syntax
func TestCreateRange(t *testing.T) {
	data := []struct {
		addrRange                  string
		expectedStart, expectedEnd int
	}{
		{"", currentLine, currentLine},
		{"1,2", 1, 2},
		{"99,999", 99, 999},
		{"9,", 9, currentLine},
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
			t.Errorf("error: %s (input string: %s)", err, test.addrRange)
		} else if r.start.addr != test.expectedStart {
			t.Errorf("bad start: %d, expected: %d (input string: %s)", r.start.addr, test.expectedStart, test.addrRange)
		} else if r.end.addr != test.expectedEnd {
			t.Errorf("bad end: %d, expected: %d (input string: %s)", r.end.addr, test.expectedEnd, test.addrRange)
		}
	}
}
