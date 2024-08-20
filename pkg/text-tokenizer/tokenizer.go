package tokenizer

import (
	"sort"

	"github.com/ikawaha/kagome/tokenizer"
)

const (
	featureTypeTokenIndex = iota
	featureTypeNounKey    = "名詞"
)

// Tokenizer is kagome.Tokenizer wrapper
type Tokenizer struct {
	t tokenizer.Tokenizer
}

// NewTokenizer initialize kagome.Tokenizer
func NewTokenizer(opt ...string) *Tokenizer {
	return &Tokenizer{
		t: tokenizer.New(),
	}
}

// Tokenize create []kagome.Token
func (m *Tokenizer) Tokenize(text string) Tokens {
	return Tokens(m.t.Tokenize(text))
}

// Tokens is tokenizer.Token wrapper
type Tokens []tokenizer.Token

// DistinctByNoun return Tokens distinct by nouns
func (m Tokens) DistinctByNoun() (tokens Tokens) {
	distinctByNouns := map[string]bool{}
	for _, token := range m {
		if token.Class == tokenizer.DUMMY {
			continue
		}
		if token.Pos() != featureTypeNounKey {
			continue
		}
		if _, ok := distinctByNouns[token.Surface]; !ok {
			distinctByNouns[token.Surface] = true
			tokens = append(tokens, token)
		}
	}
	return
}

// Sort as kagome.Token.Surface ASC
func (m Tokens) Sort() Tokens {
	sort.Sort(&m)
	return m
}

// Len is implementation of sort.Sort
func (m Tokens) Len() int {
	return len(m)
}

// Swap is implementation of sort.Sort
func (m Tokens) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

// Less is implementation of sort.Sort
func (m Tokens) Less(i, j int) bool {
	return m[i].Surface < m[j].Surface
}
