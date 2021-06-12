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
		// marks
		{"'a", "'a"},
		// regex
		{"/.*/", "/.*/"},
		{"?.*?", "?.*?"},
		// inc, dec
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
	// test empty input
	addr, err := newAddress("")
	if err != nil {
		t.Errorf("error: %s", err)
	}
	if !addr.isNotSpecified() {
		t.Errorf("expected not specified, got %s", addr)
	}

}

/*
Tests to check the calculation of the actual line number.
*/
func TestCalculateActualLineNumber(t *testing.T) {
	data := []struct {
		startLine       int
		addressStr      string
		expectedLineNbr int
	}{
		{4, "", 4}, // no-op command, stays at start line
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
		{3, "+2", 5},
		{4, "++2", 7},
		{5, "-3", 2},
		{4, "+-3", 2},
		// whitespace
		{1, "+ +", 3},
		{2, "+ ++ ++", 7},
		{3, "- -", 1},
		{6, "- -- --", 1},
		{1, "+ +2", 4},
		{2, "+ -2", 1},
		{4, "+ - ++ 2", 8},
		{1, "2++", 4},
		// . $
		{2, "$", 8},
		{3, "$-1", 7},
		{1, "$--3", 4},
		{3, ".", 3},
		{4, ".+1", 5},
		{4, ".1", 5},
		{2, ".++1", 4},
		{2, ".++4", 7},
		// mark
		{1, "'a", 2},
		// regex
		{1, "/3/", 3},
		{5, "+?3?", 3},
	}

	for _, test := range data {
		t.Run(fmt.Sprintf(">>%s<<", test.addressStr), func(t *testing.T) {
			addr, err := newAddress(test.addressStr)
			if err != nil {
				t.Errorf("error: %s", err)
			} else {
				marks := map[string]int{
					"a": 2,
				}
				lineNbr, err := addr.calculateActualLineNumber(test.startLine, createListOfLines([]string{"1", "2", "3", "4", "5", "6", "7", "8"}), marks)
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

/**
valid address but invalid actual line nbr
*/
func TestInvalidCalculateActualLineNumber(t *testing.T) {
	data := []struct {
		startLine  int
		addressStr string
	}{
		// goes past end of file
		{6, "21"},
		{2, "+7"},
		{1, "----------+18"},
		// goes before start of file
		{3, "-4"},
		// syntactically legal, but an invalid actual line
		{1, "$1"},
		{3, "1--"},
		// unknown mark
		{3, "'a+"},
	}

	for _, test := range data {
		t.Run(fmt.Sprintf(">>%s<<", test.addressStr), func(t *testing.T) {
			addr, err := newAddress(test.addressStr)
			if err != nil {
				t.Errorf("error: %s", err)
			} else {
				lineNbr, err := addr.calculateActualLineNumber(test.startLine, createListOfLines([]string{"1", "2", "3", "4", "5", "6", "7", "8"}), make(map[string]int))
				if err != nil {
					// ok
				} else {
					t.Errorf("expected error, got line nbr: %d, for address %v, starting at line %d", lineNbr, addr, test.startLine)
				}
			}
		})
	}
}

func TestMatchLineForwardOrBackward(t *testing.T) {
	data := []struct {
		forward         bool
		startLine       int
		regex           string
		expectedLineNbr int
	}{
		{true, 6, "7", 7},
		{true, 3, "[78]", 7},
		// wraparound
		{true, 3, "2", 2},
		{true, 3, "8", 8},
		// backwards search
		{false, 3, "1", 1},
		{false, 4, "[12]", 2},
		// wraparound
		{false, 3, "8", 8},
	}

	for i, test := range data {
		t.Run(fmt.Sprintf("%2d >>%s<<", i, test.regex), func(t *testing.T) {
			var lineNbr int
			var err error
			buf := createListOfLines([]string{"1", "2", "3", "4", "5", "6", "7", "8"})
			if test.forward {
				lineNbr, err = matchLineForward(test.startLine, test.regex, buf)
			} else {
				lineNbr, err = matchLineBackward(test.startLine, test.regex, buf)
			}
			if err != nil {
				t.Errorf("error: %s", err)
			} else if lineNbr != test.expectedLineNbr {
				t.Errorf("got line nbr: %d, expected %d", lineNbr, test.expectedLineNbr)
			}
		})
	}
}

/**
invalid address strings
*/
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
