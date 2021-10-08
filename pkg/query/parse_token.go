package query

import "errors"

type tokenBuffer struct {
	tokens []map[string]string
}

func NewTokenBuffer(tokens []map[string]string) *tokenBuffer {
	return &tokenBuffer{tokens: tokens}
}

// 语法分析
// 解析词法分析得到的Token，判断Token所期望的值
func (t *tokenBuffer) GrammaAnalysis() error {
	// 进行语法分析，判断语法是否正确
	stack_ := NewStack()
	flag := 0
	for index, token := range t.tokens {
		expectToken := make([]string, 0)
		for tn, _ := range token {
			switch tn {
			case TokenTypeLeftParenthesis:
				expectToken = []string{TokenTypeLeftParenthesis, TokenTypeRightParenthesis, TokenTypeBody, TokenTypeIP, TokenTypeProtocol, TokenTypeVersion, TokenTypeAttr, TokenTypeICP, TokenTypeHost, TokenTypeTitle, TokenTypeTLS, TokenTypeString}
				stack_.PUSH(TokenTypeLeftParenthesis)
				flag++
				break
			case TokenTypeRightParenthesis:
				expectToken = []string{TokenTypeRightParenthesis, TokenTypeAND, TokenTypeOR, TokenTypeString, TokenTypeEnd}
				_, err := stack_.POP()
				if err != nil {
					return err
				}
				flag--
				break
			case TokenTypeEquals, TokenTypeStrongEquals, TokenTypeNotEquals, TokenTypeRegexpEquals, TokenTypeRegexpNotEquals:
				expectToken = []string{TokenTypeString}
				break
			case TokenTypeAND, TokenTypeOR:
				expectToken = []string{TokenTypeLeftParenthesis, TokenTypeBody, TokenTypeIP, TokenTypeProtocol, TokenTypeVersion, TokenTypeAttr, TokenTypeICP, TokenTypeHost, TokenTypeTitle, TokenTypeTLS, TokenTypeString}
				break
			case TokenTypeBody, TokenTypeIP, TokenTypeProtocol, TokenTypeVersion, TokenTypeAttr, TokenTypeICP, TokenTypeHost, TokenTypeTitle, TokenTypeTLS:
				expectToken = []string{TokenTypeEquals, TokenTypeStrongEquals, TokenTypeNotEquals, TokenTypeRegexpEquals, TokenTypeRegexpNotEquals}
				break
			case TokenTypeString:
				expectToken = []string{TokenTypeRightParenthesis, TokenTypeAND, TokenTypeOR, TokenTypeEnd}
				break
			case TokenTypeStart:
				expectToken = []string{TokenTypeLeftParenthesis, TokenTypeRightParenthesis, TokenTypeBody, TokenTypeIP, TokenTypeProtocol, TokenTypeVersion, TokenTypeAttr, TokenTypeICP, TokenTypeHost, TokenTypeTitle, TokenTypeTLS, TokenTypeString}
				break
			case TokenTypeEnd:
				return nil
			}
		}
		err := t.checkToken(index, expectToken)
		if err != nil {
			return err
		}
	}
	if stack_.isEmpty() {
		return nil
	}
	return errors.New("stack not empty")
}

func (t *tokenBuffer) checkToken(index int, tokens []string) error {
	for i := 0; i < len(tokens); i++ {
		for t_, _ := range t.tokens[index+1] {
			if t_ == tokens[i] {
				return nil
			}
		}
	}
	current := ""
	for _, v := range t.tokens[index] {
		current = v
	}
	next := ""
	for _, v := range t.tokens[index+1] {
		next = v
	}
	return errors.New(current + " expect token error, not " + next)
}
