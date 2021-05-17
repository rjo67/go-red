package red

import (
	"fmt"
	"testing"
)

/**
Tests to check the internal structure after parsing an address line
*/
func TestRawParseAddress(t *testing.T) {
	data := []struct {
		addressStr string
		expected   string
	}{
		{"", ""},
		// marks
		{"'a", "'a"},
		{"/.*/", "/.*/"},
		{"?.*?", "?.*?"},
		{"+", "+"},
		{"++", "+,+"},
		{"+++++", "+,+,+,+,+"},
		{"-", "-"},
		{"---", "-,-,-"},
		{"-----", "-,-,-,-,-"},
		{"++---", "+,+,-,-,-"},
		{"+++--", "+,+,+,-,-"},
		{"+++--+++", "+,+,+,-,-,+,+,+"},
		{"--++", "-,-,+,+"},
		{"3", "3"},
		{"++-+3", "+,+,-,+3"},
		{"21", "21"},
		{"+2", "+2"},
		{"++2", "+,+2"},
		{"+23", "+23"},
		{"----------+23", "-,-,-,-,-,-,-,-,-,-,+23"},
		{"-3", "-3"},
		{"+-3", "+,-3"},
		{"-31", "-31"},
		// whitespace
		{"+ +", "+,+"},
		{"+ ++ ++", "+,+,+,+,+"},
		{"- -", "-,-"},
		{"- -- --", "-,-,-,-,-"},
		{"+ +2", "+,+2"},
		{"+ -2", "+,-2"},
		{"+ - ++ 2", "+,-,+,+,2"},
		// . $
		{"$", "$"},
		{"$-1", "$,-1"},
		{"$--3", "$,-,-3"},
		{".", "."},
		{".+1", ".,+1"},
		{".1", ".,1"},
		{".++1", ".,+,+1"},
		{".++4", ".,+,+4"},
		{"$1", "$,1"}, // syntactically legal, although an invalid range
		{".1", ".,1"},
		{"2++", "2,+,+"},
	}
	for _, test := range data {
		t.Run(fmt.Sprintf(">>%s<<", test.addressStr), func(t *testing.T) {
			addr, err := newAddress(test.addressStr)
			if err != nil {
				t.Errorf("error: %s", err)
			}
			addressParts := addr.addressPartsAsString()
			if addressParts != test.expected {
				t.Errorf("wrong result, got: %s, expected: %s", addressParts, test.expected)
			}
		})
	}
}

func TestCalculateActualLineNumber(t *testing.T) {
	data := []struct {
		startLine       int
		addressStr      string
		expectedLineNbr int
	}{
		//{"", notSpecified, 0},
		{1, "+", 2},
		{2, "++", 4},
		{1, "+++++", 6},
		{2, "-", 1},
		{7, "---", 4},
		{8, "-----", 3},
		{5, "++---", 4},
		{8, "++---", 7}, // temporarily goes 'over' end of buffer
		{3, "+++--", 4},
		{1, "--+++--+", 1}, // temporarily goes 'below' start of buffer
		{2, "+++--+++", 6},
		{3, "--++", 3},
		{2, "3", 3},
		{3, "++-+3", 7},
		//{"21", 21, 0},
		{3, "+2", 5},
		{4, "++2", 7},
		//{"+23", currentLine, 23},
		//{"----------+23", currentLine, 13},
		{5, "-3", 2},
		{4, "+-3", 2},
		//{"-31", currentLine, -31},
		// whitespace
		{1, "+ +", 3},
		{2, "+ ++ ++", 7},
		{3, "- -", 1},
		{6, "- -- --", 1},
		{1, "+ +2", 4},
		{2, "+ -2", 1},
		{4, "+ - ++ 2", 8},
		// . $
		{2, "$", 8},
		{3, "$-1", 7},
		{1, "$--3", 4},
		{3, ".", 3},
		{4, ".+1", 5},
		{4, ".1", 5},
		{2, ".++1", 4},
		{2, ".++4", 7},
		//{"$1", endOfFile, 1}, // syntactically legal, although an invalid range
		{1, "2++", 4},
	}

	for _, test := range data {
		t.Run(fmt.Sprintf(">>%s<<", test.addressStr), func(t *testing.T) {
			addr, err := newAddress(test.addressStr)
			if err != nil {
				t.Errorf("error: %s", err)
			} else {
				lineNbr, err := addr.calculateActualLineNumber2(test.startLine, createListOfLines([]string{"1", "2", "3", "4", "5", "6", "7", "8"}))
				if err != nil {
					t.Errorf("error: %s", err)
				} else {
					if lineNbr != test.expectedLineNbr {
						t.Errorf("wrong line nbr, got: %d, expected: %d, for address %v, starting at line %d", lineNbr, test.expectedLineNbr, addr, test.startLine)
					}
				}
			}
		})
	}
}

func TestAddressErrors(t *testing.T) {
	data := []struct {
		addressStr string
	}{
		{"'qwe"},  // mark can only be one letter
		{"/we"},   // non-terminated regex
		{"?we  "}, // non-terminated regex wich spaces
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
