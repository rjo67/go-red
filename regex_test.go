package red

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestSubstitute(t *testing.T) {
	state := State{}
	state.Buffer = createListOfLines([]string{"rjo", "rjo", "my name is rjo", "bar"})

	// to capture the output
	var buff bytes.Buffer // implements io.Writer

	nbrLinesChanged, _, err := processLines(&buff, 2, state.Buffer.Len(), &state, "rjo", "foobar", "gp")
	if err != nil {
		t.Fatalf("error %s", err)
	}
	if nbrLinesChanged != 2 {
		t.Fatalf("wrong number of lines changed, expected %d but got %d", 2, nbrLinesChanged)
	}
	if buff.String() != "foobar\nmy name is foobar\n" {
		t.Fatalf("changed lines '%s'", buff.String())
	}

}

func TestFindNamedMatches(t *testing.T) {
	//re := regexp.MustCompile(`(?P<special>[\.\$ ]|'[a-z]|\/.*\/|\?.*\?|[+-]?\d*|[-+])`)
	re := regexp.MustCompile(`(?P<special>[\.\$])|(?P<mark>'[a-z])|(?P<reFor>\/[^/]*\/)|(?P<reBack>\?[^\?]*\?)|(?P<signednbr>[+-]?\d+)|(?P<incdec>[-+])`)
	//	re := regexp.MustCompile(`[\.\$]|'[a-z]|\/[^/]*\/|\?[^\?]*\?|[+-]?\d+|[-+]`)

	//str := `'a'b +/345 99-.$ ?24? + 5 -3--` // unmatched regex : finds a signednbr instead (345)
	//str := `'a /345 ?24? /.*/` // unmatched regex
	//	str := `./.* ' a/+2$?24+2?+--2+ 2` // unmatched regex
	str := `/qwe  sdfsd` // unmatched regex

	//fmt.Printf("namedmatch: %v\n", findNamedMatches(re, str))
	//fmt.Printf("all NamedMatches: %v\n", findAllNamedMatches(re, str, false))

	index := 0
	for loc := re.FindStringIndex(str); loc != nil; loc = re.FindStringIndex(str[index:]) {
		startOfMatch := index + loc[0]
		if loc[0] != 0 {
			// char(s) were skipped during the previous call of FindStringIndex -- either spaces or error
			trimmed := strings.TrimSpace(str[index:startOfMatch])
			if len(trimmed) != 0 {
				locnString, errorDesc := analyzeParseError(index, startOfMatch, trimmed)
				t.Fatalf("%s at posn %s: >>%s<<", errorDesc, locnString, str[index:startOfMatch])
			}
		}
		fmt.Printf("%02d-%02d >>%s<<\n", startOfMatch, index+loc[1], str[startOfMatch:index+loc[1]])
		index += loc[1]
	}
	if index == 0 {
		fmt.Println("no match")
	}
	//t.Fail()
}
