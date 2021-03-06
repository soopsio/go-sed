package lexer

import (
	"fmt"
	"testing"

	"github.com/zkry/go-sed/token"
)

func TestReadChar(t *testing.T) {
	cases := []struct {
		title            string
		program          string
		readCharCt       int
		expCh, expPrevCh rune
	}{
		{
			title:      "Basic test of read char",
			program:    "/addr/s/one/two/g",
			readCharCt: 5,
			expCh:      '/',
			expPrevCh:  'r',
		},
		{
			title:      "Test starting conditions",
			program:    "/addr/s/one/two/g",
			readCharCt: 0,
			expCh:      '/',
			expPrevCh:  0,
		},
		{
			title:      "Test reading past last character",
			program:    "123",
			readCharCt: 3,
			expCh:      0,
			expPrevCh:  '3',
		},
	}

	for i, c := range cases {
		l := New(c.program)
		for j := 0; j < c.readCharCt; j++ {
			l.readChar()
		}
		if l.ch != c.expCh || l.prevCh != c.expPrevCh {
			t.Errorf("Test %d (%s) failed.\n  exp ch=%v, got %v\n  exp prevCh=%v, got %v\n",
				i, c.title, c.expCh, l.ch, c.expPrevCh, l.prevCh)
		}
	}
}

func TestReadUntil(t *testing.T) {
	cases := []struct {
		title            string
		preReadChar      int
		program          string
		toFunc           func(rune) bool
		expCh, expPrevCh rune
		expRes           string
	}{
		{
			title:       "Reading until the next /",
			program:     "one two three/",
			preReadChar: 0,
			toFunc:      func(r rune) bool { return r == '/' },
			expCh:       '/',
			expPrevCh:   'e',
			expRes:      "one two three",
		},
		{
			title:       "Reading until the next / between / /",
			program:     "/one two three/",
			preReadChar: 1,
			toFunc:      func(r rune) bool { return r == '/' },
			expCh:       '/',
			expPrevCh:   'e',
			expRes:      "one two three",
		},
		{
			title:       "Reading starting of on the symbol reading to",
			program:     "/one two three/",
			preReadChar: 0,
			toFunc:      func(r rune) bool { return r == '/' },
			expCh:       '/',
			expPrevCh:   0,
			expRes:      "",
		},
		{
			title:       "Escapeing the toFunc",
			program:     "one \\/two \\/three/",
			preReadChar: 0,
			toFunc:      func(r rune) bool { return r == '/' },
			expCh:       '/',
			expPrevCh:   'e',
			expRes:      `one /two /three`,
		},
		{
			title:       "The end is never found",
			program:     "one two three four ...",
			preReadChar: 0,
			toFunc:      func(r rune) bool { return r == '/' },
			expCh:       0,
			expPrevCh:   '.',
			expRes:      `one two three four ...`,
		},
		{
			title:       "Escaping the very end?",
			program:     "one two three four ...\\",
			preReadChar: 0,
			toFunc:      func(r rune) bool { return r == '/' },
			expCh:       0,
			expPrevCh:   '\\',
			expRes:      `one two three four ...\`,
		},
		{
			title:       "Escaping escape chars",
			program:     `\\ \\ \\/this shouldn't be returned`,
			preReadChar: 0,
			toFunc:      func(r rune) bool { return r == '/' },
			expCh:       '/',
			expPrevCh:   '\\',
			expRes:      `\\ \\ \\`,
		},
		{
			title:       "Escaping escape chars",
			program:     `one two \ three/`,
			preReadChar: 0,
			toFunc:      func(r rune) bool { return r == '/' },
			expCh:       '/',
			expPrevCh:   'e',
			expRes:      `one two \ three`,
		},
	}

	for i, c := range cases {
		l := New(c.program)
		for j := 0; j < c.preReadChar; j++ {
			l.readChar()
		}
		res := l.readUntilEscape(c.toFunc)
		if l.ch != c.expCh || l.prevCh != c.expPrevCh || res != c.expRes {
			t.Errorf("Test %d (%s) failed.\n  exp res='%v', got='%v'\n  exp ch=%v(%c), got=%v(%c)\n  exp prevCh=%v(%c), got=%v(%c)\n", i, c.title, c.expRes, res, c.expCh, c.expCh, l.ch, l.ch, c.expPrevCh, c.expPrevCh, l.prevCh, l.prevCh)
		}
	}
}

func TestNextTokens(t *testing.T) {
	for i, lt := range lexerTests {
		l := New(lt.program)
		for j, et := range lt.expected {
			gotTok := l.NextToken()

			if gotTok.Type != et.Type {
				t.Errorf("Program[%d]:%s line[%d] - tokentype wrong. expected=%v, got=%v", i, lt.program, j, et.Type, gotTok.Type)
			}

			if gotTok.Literal != et.Literal {
				t.Fatalf("Program[%d]:%s line[%d] - tokenliteral wrong. expected=%v, got=%v", i, lt.program, j, et.Literal, gotTok.Literal)
			}
		}
	}
}

func TestNextToken(t *testing.T) {
	cases := []struct {
		t      string
		prg    string
		s      state
		expTok token.Token
		expCh  rune
	}{
		{t: "simple start", prg: "s/one/two/g", s: stateStart, expTok: tok(token.CMD, "s"), expCh: '/'},
		{t: "space at beginning", prg: "    s/one/two/g", s: stateStart, expTok: tok(token.CMD, "s"), expCh: '/'},
		{t: "addr at beginning", prg: "    /addr/s/one/two/g", s: stateStart, expTok: tok(token.SLASH, "/"), expCh: 'a'},
		{t: "blank addr at beginning", prg: "    //s/one/two/g", s: stateStart, expTok: tok(token.SLASH, "/"), expCh: '/'},
		{t: "Escape address", prg: `\s\ss,\s\sss\ss\s\ss`, s: stateStart, expTok: tok(token.SLASH, "s"), expCh: '\\'},
		{t: "Number at start", prg: "123,$s/a/b/", s: stateStart, expTok: tok(token.INT, "123"), expCh: ','},
		{t: "Illegal at start", prg: "***", s: stateStart, expTok: tok(token.ILLEGAL, "*"), expCh: '*'},
	}

	for i, c := range cases {
		fmt.Println("------")
		lexer := New(c.prg)
		lexer.s = c.s
		gotToken := lexer.NextToken()
		fmt.Printf("->'%c'\n", lexer.ch)

		if !tokEqu(gotToken, c.expTok) {
			t.Errorf("Test %d (%s) incorrect token:\n  Expected: %v, got: %v\n", i, c.t, gotToken, c.expTok)
		}
		if lexer.ch != c.expCh {
			t.Errorf("Test %d (%s) incorrect subsequent position:\n  Expected: %v(%c), got: %v(%c)\n", i, c.t, c.expCh, c.expCh, lexer.ch, lexer.ch)
		}
	}
}

func tok(t token.Type, l string) token.Token {
	return token.Token{Type: t, Literal: l}
}

func tokEqu(t1, t2 token.Token) bool {
	return t1.Type == t2.Type && t1.Literal == t2.Literal
}

var lexerTests = []struct {
	program  string
	expected []token.Token
}{
	{ // Program 0
		program: "/",
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 1
		program: "/addr/",
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 2
		program: "/addr1/,/addr2/",
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr1"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr2"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 3
		program: "/addr1/,/addr2/d",
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr1"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr2"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 4
		program: "/addr1/,/addr2/s/find/replace/",
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr1"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr2"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "find"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "replace"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 5
		program: "/addr1/,/addr2/s/find/replace/g",
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr1"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr2"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "find"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "replace"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "g"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 6
		program: `/-> addr1 <-/,/!@#$%\/*+/s/some text/~~~~~~/g`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "-> addr1 <-"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: `!@#$%\/*+`},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "some text"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "~~~~~~"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "g"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 7
		program: `s/one/two/`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 8
		program: `s/one/two/p`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "p"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 9
		program: `y/abc/xyz/`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "y"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "abc"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "xyz"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 10
		program: `/addr/d`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 11
		program: `/addr/ d`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 12
		program: `/addr/     d`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 13
		program: `/addr1/,/addr2/     d`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr1"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr2"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 14
		program: `/addr1/,/addr2/s/one/two/w outfile.txt`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr1"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr2"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "w"},
			token.Token{Type: token.IDENT, Literal: "outfile.txt"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 15
		program: `/addr1/,/addr2/s/one/two/w      outfile.txt`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr1"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr2"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "w"},
			token.Token{Type: token.IDENT, Literal: "outfile.txt"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 16
		program: `/addr1/,/addr2/s/one/two/woutfile.txt`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr1"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr2"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "w"},
			token.Token{Type: token.IDENT, Literal: "outfile.txt"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 17
		program: `/addr1/,/addr2/s/one/two/woutfile.txt`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr1"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr2"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "w"},
			token.Token{Type: token.IDENT, Literal: "outfile.txt"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 18
		program: `s/one/two/
s/three/four/`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "three"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "four"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 19
		program: `s/one/two/p
s/three/four/
s/five/six/p`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "p"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "three"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "four"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "five"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "six"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "p"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 20
		program: `s/one/two/p
/addr1/,/addr2/s/three/four/
s/five/six/p`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "p"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr1"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "addr2"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "three"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "four"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "five"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "six"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "p"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 21
		program: `$d`,
		expected: []token.Token{
			token.Token{Type: token.DOLLAR, Literal: "$"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 22
		program: `5d`,
		expected: []token.Token{
			token.Token{Type: token.INT, Literal: "5"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 23
		program: `1,5d`,
		expected: []token.Token{
			token.Token{Type: token.INT, Literal: "1"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.INT, Literal: "5"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 24
		program: `5,$d`,
		expected: []token.Token{
			token.Token{Type: token.INT, Literal: "5"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.DOLLAR, Literal: "$"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 25
		program: `5,$  d`,
		expected: []token.Token{
			token.Token{Type: token.INT, Literal: "5"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.DOLLAR, Literal: "$"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 26
		program: `5,$  d
1,2d
3,4d
s/a/b/p`,
		expected: []token.Token{
			token.Token{Type: token.INT, Literal: "5"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.DOLLAR, Literal: "$"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.INT, Literal: "1"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.INT, Literal: "2"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.INT, Literal: "3"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.INT, Literal: "4"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "a"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "b"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "p"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 27
		program: `s|a|b|`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.LIT, Literal: "a"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.LIT, Literal: "b"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 28
		program: `s|a|b|p`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.LIT, Literal: "a"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.LIT, Literal: "b"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.IDENT, Literal: "p"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 29
		program: `s,a,b,r file.io`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: ","},
			token.Token{Type: token.LIT, Literal: "a"},
			token.Token{Type: token.DIV, Literal: ","},
			token.Token{Type: token.LIT, Literal: "b"},
			token.Token{Type: token.DIV, Literal: ","},
			token.Token{Type: token.IDENT, Literal: "r"},
			token.Token{Type: token.IDENT, Literal: "file.io"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 30
		program: `100,/funny/s,a,b,b`,
		expected: []token.Token{
			token.Token{Type: token.INT, Literal: "100"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "funny"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: ","},
			token.Token{Type: token.LIT, Literal: "a"},
			token.Token{Type: token.DIV, Literal: ","},
			token.Token{Type: token.LIT, Literal: "b"},
			token.Token{Type: token.DIV, Literal: ","},
			token.Token{Type: token.IDENT, Literal: "b"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 31
		program: `s/delete me//`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "delete me"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 32
		program: `s///`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 33
		program: `s////`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.ILLEGAL, Literal: "/"},
		},
	},
	{ // Program 34
		program: `$,$,`,
		expected: []token.Token{
			token.Token{Type: token.DOLLAR, Literal: "$"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.DOLLAR, Literal: "$"},
			token.Token{Type: token.ILLEGAL, Literal: ","},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 35
		program: `/WORD/ i\
Add this line before every line with WORD`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "WORD"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "i"},
			token.Token{Type: token.BACKSLASH, Literal: "\\"},
			token.Token{Type: token.LIT, Literal: "Add this line before every line with WORD"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 36
		program: `/WORD/ c\
Replace the current line with the line`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "WORD"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "c"},
			token.Token{Type: token.BACKSLASH, Literal: "\\"},
			token.Token{Type: token.LIT, Literal: "Replace the current line with the line"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 37
		program: `
s/blank/lines/`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "blank"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "lines"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 38
		program: `# This is a comment
s/blank/lines/`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "#"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "blank"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "lines"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 39
		program: `    # This is a comment
s/blank/lines/`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "#"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "blank"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "lines"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 40
		program: `3 s/[0-9][0-9]*//`,
		expected: []token.Token{
			token.Token{Type: token.INT, Literal: "3"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "[0-9][0-9]*"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 41
		program: `/^#/ s/[0-9][0-9]*//`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "^#"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "[0-9][0-9]*"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 42
		program: `/^#/ s/[0-9][0-9]*//`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "^#"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "[0-9][0-9]*"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 43
		program: `\_/usr/local/bin_ s_/usr/local_/common/all_`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "_"},
			token.Token{Type: token.LIT, Literal: "/usr/local/bin"},
			token.Token{Type: token.SLASH, Literal: "_"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "_"},
			token.Token{Type: token.LIT, Literal: "/usr/local"},
			token.Token{Type: token.DIV, Literal: "_"},
			token.Token{Type: token.LIT, Literal: "/common/all"},
			token.Token{Type: token.DIV, Literal: "_"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 44
		program: `/^g/ s_g_s_g`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "^g"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "_"},
			token.Token{Type: token.LIT, Literal: "g"},
			token.Token{Type: token.DIV, Literal: "_"},
			token.Token{Type: token.LIT, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "_"},
			token.Token{Type: token.IDENT, Literal: "g"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: `1,100 s/A/a/`,
		expected: []token.Token{
			token.Token{Type: token.INT, Literal: "1"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.INT, Literal: "100"},

			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "A"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "a"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: `p
p
p`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "p"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.CMD, Literal: "p"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.CMD, Literal: "p"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: "d",
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: " p",
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "p"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: "\tp",
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "p"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: `
	/begin/n
	s/old/new/`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "begin"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "n"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "old"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "new"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: `# Testing Grouping
/begin/,/end/ {
s/#.*//
	s/[ ^I]*$//
	/^$/ d
	p
}`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "#"},

			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "begin"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "end"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LBRACE, Literal: "{"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "#.*"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "[ ^I]*$"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "^$"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "p"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.RBRACE, Literal: "}"},

			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: `
	1,100 {
		/begin/,/end/ {
		     s/#.*//
		     s/[ ^I]*$//
		     /^$/ d
		     p
		}
	}`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.INT, Literal: "1"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.INT, Literal: "100"},
			token.Token{Type: token.LBRACE, Literal: "{"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "begin"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "end"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LBRACE, Literal: "{"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "#.*"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "[ ^I]*$"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "^$"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "p"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.RBRACE, Literal: "}"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.RBRACE, Literal: "}"},

			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: `
	1,100 !{
		/begin/,/end/ !{
		     s/#.*//
		     s/[ ^I]*$//
		     /^$/ d
		     p
		}
	}`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.INT, Literal: "1"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.INT, Literal: "100"},
			token.Token{Type: token.EXPLMARK, Literal: "!"},
			token.Token{Type: token.LBRACE, Literal: "{"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "begin"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "end"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.EXPLMARK, Literal: "!"},
			token.Token{Type: token.LBRACE, Literal: "{"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "#.*"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "[ ^I]*$"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "^$"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "p"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.RBRACE, Literal: "}"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.RBRACE, Literal: "}"},

			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: `
	1,100!{
		/begin/,/end/ !{
			/begin/n
			s/old/new/
		}
	}`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.INT, Literal: "1"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.INT, Literal: "100"},
			token.Token{Type: token.EXPLMARK, Literal: "!"},
			token.Token{Type: token.LBRACE, Literal: "{"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.SLASH, Literal: "/"}, // 7
			token.Token{Type: token.LIT, Literal: "begin"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.COMMA, Literal: ","},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "end"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.EXPLMARK, Literal: "!"},
			token.Token{Type: token.LBRACE, Literal: "{"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.SLASH, Literal: "/"}, // 17
			token.Token{Type: token.LIT, Literal: "begin"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "n"},
			token.Token{Type: token.NEWLINE, Literal: "\n"}, // 21

			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "old"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "new"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.RBRACE, Literal: "}"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.RBRACE, Literal: "}"},

			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: `
/^$/ bpara
H
$ bpara
b
:para
x
/'$1'/ p
`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.SLASH, Literal: "/"}, // 7
			token.Token{Type: token.LIT, Literal: "^$"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "b"},
			token.Token{Type: token.IDENT, Literal: "para"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "H"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.DOLLAR, Literal: "$"},
			token.Token{Type: token.CMD, Literal: "b"},
			token.Token{Type: token.IDENT, Literal: "para"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "b"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.COLON, Literal: ":"},
			token.Token{Type: token.IDENT, Literal: "para"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "x"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.SLASH, Literal: "/"}, // 7
			token.Token{Type: token.LIT, Literal: "'$1'"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "p"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: `
:again
	s/([ ^I]*)//
	tagain
`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.COLON, Literal: ":"},
			token.Token{Type: token.IDENT, Literal: "again"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "([ ^I]*)"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.CMD, Literal: "t"},
			token.Token{Type: token.IDENT, Literal: "again"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: `
/grep/ !{;H;x;s/^.*\n\(.*\n.*\)$/\1/;x;}
/grep/ {;H;n;H;x;p;a\
---
}
`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "grep"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.EXPLMARK, Literal: "!"},
			token.Token{Type: token.LBRACE, Literal: "{"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "H"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "x"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "^.*\\n\\(.*\\n.*\\)$"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "\\1"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "x"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.RBRACE, Literal: "}"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "grep"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LBRACE, Literal: "{"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "H"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "n"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "H"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "x"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "p"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "a"},
			token.Token{Type: token.BACKSLASH, Literal: "\\"},
			token.Token{Type: token.LIT, Literal: "---"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},

			token.Token{Type: token.RBRACE, Literal: "}"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 45
		program: `
a \
---
`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.CMD, Literal: "a"},
			token.Token{Type: token.BACKSLASH, Literal: "\\"},
			token.Token{Type: token.LIT, Literal: "---"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 46
		program: `s|a|b|g|`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.LIT, Literal: "a"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.LIT, Literal: "b"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.IDENT, Literal: "g"},
			token.Token{Type: token.ILLEGAL, Literal: "|"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 47
		program: `s|a|b|gp`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.LIT, Literal: "a"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.LIT, Literal: "b"},
			token.Token{Type: token.DIV, Literal: "|"},
			token.Token{Type: token.IDENT, Literal: "g"},
			token.Token{Type: token.IDENT, Literal: "p"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 48
		program: `s/one/two/;s/two/three/;`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "three"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 49
		program: "s/one/two/\ns/two/three/",
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "three"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{ // Program 48
		program: `s/one/two;`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "one"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "two;"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{
		program: `$s/$/}/

/./!d`,
		expected: []token.Token{
			token.Token{Type: token.DOLLAR, Literal: "$"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "$"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "}"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "."},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.EXPLMARK, Literal: "!"},
			token.Token{Type: token.CMD, Literal: "d"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{
		program: `/a/b branch`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "a"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.CMD, Literal: "b"},
			token.Token{Type: token.IDENT, Literal: "branch"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{
		program: `s/.*/\
|--------|\
|        |\
|        |\
|--------|/`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: ".*"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "\n|--------|\n|        |\n|        |\n|--------|"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{
		program: `
  /^t3$/{ s/.*/\
 TEST 3 - 3\
      _____________ \
     |     ==      |\
     |     ==      |\
     |    ==  =    |\
     |     = ==    |\
     |  =o         |\
     |  ==         |\
     |             |\
     |.____________|\
   / ; b endmap
  }`,
		expected: []token.Token{
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LIT, Literal: "^t3$"},
			token.Token{Type: token.SLASH, Literal: "/"},
			token.Token{Type: token.LBRACE, Literal: "{"},
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: ".*"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: `
 TEST 3 - 3
      _____________ 
     |     ==      |
     |     ==      |
     |    ==  =    |
     |     = ==    |
     |  =o         |
     |  ==         |
     |             |
     |.____________|
   `},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			// Fix random semicolon insertion
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.CMD, Literal: "b"},
			token.Token{Type: token.IDENT, Literal: "endmap"},
			token.Token{Type: token.NEWLINE, Literal: "\n"},
			token.Token{Type: token.RBRACE, Literal: "}"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{
		program: `s/\\\\/\\/g`,
		expected: []token.Token{
			token.Token{Type: token.CMD, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: `\\\\`},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.LIT, Literal: `\\`},
			token.Token{Type: token.DIV, Literal: "/"},
			token.Token{Type: token.IDENT, Literal: "g"},
			token.Token{Type: token.EOF, Literal: ""},
		},
	},
	{
		program: `\s\ss,\s\ssss\ss\s\sswsss`,
		expected: []token.Token{
			token.Token{Type: token.SLASH, Literal: "s"},
			token.Token{Type: token.SLASH, Literal: "s"},
			token.Token{Type: token.SEMICOLON, Literal: ";"},
			token.Token{Type: token.SLASH, Literal: "s"},
			token.Token{Type: token.SLASH, Literal: "s"},

			token.Token{Type: token.CMD, Literal: "s"},

			token.Token{Type: token.DIV, Literal: "s"},
			token.Token{Type: token.LIT, Literal: "s"},
			token.Token{Type: token.DIV, Literal: "s"},
			token.Token{Type: token.LIT, Literal: "ss"},
			token.Token{Type: token.DIV, Literal: "s"},

			token.Token{Type: token.IDENT, Literal: "w"},
			token.Token{Type: token.IDENT, Literal: "sss"},

			token.Token{Type: token.EOF, Literal: ""},
		},
	},
}
