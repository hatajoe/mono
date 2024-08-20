package tokenizer

import (
	"fmt"
	"os"
	"testing"
)

var myTokenizer *Tokenizer

func TestMain(m *testing.M) {
	var err error
	myTokenizer = NewTokenizer()
	if err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestTokenizeAsNoun1(t *testing.T) {
	nouns := myTokenizer.Tokenize("寿司食べたい。").DistinctByNoun().Sort()
	if len(nouns) != 1 {
		t.Errorf("err: length of nouns is expecting 1")
	}
	if nouns[0].Surface != "寿司" {
		t.Errorf("err: nonus is %s expecting `寿司`", nouns[0])
	}
}

func TestTokenizeAsNoun2(t *testing.T) {
	nouns := myTokenizer.Tokenize("寿司と焼き肉食べたい。").DistinctByNoun().Sort()
	if len(nouns) != 2 {
		t.Error("err: length of nouns is expecting 2")
	}
	if nouns[0].Surface != "寿司" {
		t.Errorf(fmt.Sprintf("err: nonus is %s expecting `寿司`", nouns[0]))
	}
	if nouns[1].Surface != "焼き肉" {
		t.Errorf(fmt.Sprintf("err: nonus is %s expecting `焼き肉`", nouns[1]))
	}
}

func TestTokenizeAsNoun3(t *testing.T) {
	nouns := myTokenizer.Tokenize("すもももももももものうち。").DistinctByNoun().Sort()
	if len(nouns) != 3 {
		t.Error("err: length of nouns is expecting 3")
	}
	if nouns[0].Surface != "うち" {
		t.Errorf(fmt.Sprintf("err: nonus is %s expecting `うち`", nouns[0]))
	}
	if nouns[1].Surface != "すもも" {
		t.Errorf(fmt.Sprintf("err: nonus is %s expecting `すもも`", nouns[1]))
	}
	if nouns[2].Surface != "もも" {
		t.Errorf(fmt.Sprintf("err: nonus is %s expecting `もも`", nouns[2]))
	}
}

func BenchmarkTokenizeAsNoun(b *testing.B) {
	for i := 0; i < b.N; i++ {
		myTokenizer.Tokenize("すもももももももものうち。").DistinctByNoun().Sort()
	}
}
