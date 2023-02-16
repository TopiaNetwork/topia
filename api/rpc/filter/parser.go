package filter

import (
	"errors"
	"strings"
)

type parser struct {
	step         step
	i            int
	filterString string
}

type step int

const (
	stepStart step = iota
	stepFieldName
	stepOperator
	stepData
	stepLogicOp // OR AND
)

func (p *parser) parseFilterString(eventName string, filterString string) (*filter, error) {
	// TODO 检查 filterString 长度

	if len(strings.TrimSpace(filterString)) == 0 {
		return &filter{eventName: eventName, directPass: true}, nil
	}

	p.filterString = filterString

	p.popWhitespace() // pop whitespace of the string's beginning

	f := &filter{
		eventName:  eventName,
		clauses:    [][]condition{{}},
		directPass: false,
	}

	for {
		if p.i >= len(p.filterString) {
			return f, nil
		}

		switch p.step {
		case stepStart: // start
			p.step = stepFieldName
		case stepFieldName:
			fieldName := p.peek()
			fieldName = strings.ToLower(fieldName)
			f.clauses[len(f.clauses)-1] = append(f.clauses[len(f.clauses)-1], condition{field: fieldName})
			p.pop()
			p.step = stepOperator
		case stepOperator:
			opString := p.peek()
			opString = strings.ToLower(opString)
			op, ok := stringToOp[opString]
			if !ok {
				return nil, errors.New("illegal operator: " + opString)
			}
			f.clauses[len(f.clauses)-1][len(f.clauses[len(f.clauses)-1])-1].operator = op
			p.pop()
			if op == Exists {
				p.step = stepLogicOp
			} else {
				p.step = stepData
			}
		case stepData:
			dataString := p.peek()
			f.clauses[len(f.clauses)-1][len(f.clauses[len(f.clauses)-1])-1].data = dataString
			p.pop()
			p.step = stepLogicOp
		case stepLogicOp:
			logicOpString := p.peek()
			logicOpString = strings.ToLower(logicOpString)
			if logicOpString != "or" && logicOpString != "and" {
				return nil, errors.New("illegal logic operator: " + logicOpString)
			}

			if logicOpString == "or" {
				f.clauses = append(f.clauses, []condition{})
			}

			p.pop()
			p.step = stepFieldName

		default:
			return nil, errors.New("unexpected step")

		}

	}

}

func (p *parser) peek() string {
	peeked, _ := p.peekWithLength()
	return peeked
}

func (p *parser) pop() string {
	peeked, length := p.peekWithLength()
	p.i += length
	p.popWhitespace()
	return peeked
}

func (p *parser) peekWithLength() (string, int) {
	if p.i >= len(p.filterString) {
		return "", 0
	}

	for _, rWord := range reservedSigns {
		token := p.filterString[p.i:min(len(p.filterString), p.i+len(rWord))]
		if token == rWord {
			return token, len(token)
		}
	}

	if p.filterString[p.i] == '\'' { // Quoted string
		return p.peekQuotedStringWithLength()
	}
	return p.peekToNextSpaceOrReservedSign()
}

func (p *parser) peekQuotedStringWithLength() (string, int) {
	if p.i > len(p.filterString) || p.filterString[p.i] != '\'' {
		return "", 0
	}
	for i := p.i + 1; i < len(p.filterString); i++ {
		if p.filterString[i] == '\'' && p.filterString[i-1] != '\\' {
			return p.filterString[p.i+1 : i], len(p.filterString[p.i+1:i]) + 2 // +2 for the two quotes
		}
	}
	return "", 0
}

func (p *parser) peekToNextSpace() (string, int) {
	for i := p.i; i < len(p.filterString); i++ {
		if p.filterString[i] == ' ' {
			return p.filterString[p.i:i], len(p.filterString[p.i:i])
		}
	}
	return p.filterString[p.i:], len(p.filterString[p.i:])
}

func (p *parser) peekToNextSpaceOrReservedSign() (string, int) {
	for i := p.i; i < len(p.filterString); i++ {
		if p.filterString[i] == ' ' {
			return p.filterString[p.i:i], len(p.filterString[p.i:i])
		}

		for _, rWord := range reservedSigns {
			token := p.filterString[i:min(len(p.filterString), i+len(rWord))]
			if token == rWord {
				return p.filterString[p.i:i], len(p.filterString[p.i:i])
			}
		}
	}
	return p.filterString[p.i:], len(p.filterString[p.i:])
}

func (p *parser) popWhitespace() {
	for ; p.i < len(p.filterString) && p.filterString[p.i] == ' '; p.i++ {
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
