package query

import "errors"

const (
	TokenTypeLeftParenthesis  = "LeftParenthesis"  // (
	TokenTypeRightParenthesis = "RightParenthesis" // )
	TokenTypeEquals           = "Equals"           // LIKE ?
	TokenTypeStrongEquals     = "StrongEquals"     // =
	TokenTypeNotEquals        = "NotEquals"        // NOT LIKE ?
	TokenTypeRegexpEquals     = "RegexpEquals"     // REGEXP
	TokenTypeRegexpNotEquals  = "RegexpNotEquals"  // NOT REGEXP ?
	TokenTypeAND              = "AND"              // AND
	TokenTypeOR               = "OR"               // OR
	TokenTypeString           = "String"           // ?
	TokenTypeSpace            = "Space"            // 空格
	TokenTypeError            = "Error"
	TokenTypeStart            = "Start"
	TokenTypeEnd              = "End"
	TokenTypeIP               = "IP"       // `ip`
	TokenTypeProtocol         = "Protocol" // `protocol`
	TokenTypeVersion          = "Version"  // `version`
	TokenTypeAttr             = "attr"     // `attr`
	TokenTypeHost             = "Host"     // `host`
	TokenTypeTitle            = "Title"    // `title`
	TokenTypeTLS              = "TLS"      // `tls`
	TokenTypeICP              = "ICP"      // `icp`
	TokenTypeBody             = "Body"     // `body`
)

type strbuffer struct {
	input []rune // 未用[]byte, 防止中文乱码
	index int
}

func (sb *strbuffer) Next() (str string, end bool) {
	if sb.index < len(sb.input) {
		str = string(sb.input[sb.index])
		sb.index++
		return
	}
	return "", true
}

func (sb *strbuffer) Reduce() {
	sb.index--
}

func (sb *strbuffer) DeleteSpace() {
	for {
		n, e := sb.Next()
		if e {
			break
		}
		if n != " " {
			sb.Reduce()
			break
		}
	}
}

// REF: https://segmentfault.com/a/1190000010998941
// 词法分析
// 逐字符读取，判断期望值
func (sb *strbuffer) LexicalAnalysis() (map[string]string, error) {
	ret := make(map[string]string)
	next, end := sb.Next()
	if end {
		ret[TokenTypeEnd] = ""
		return ret, nil
	}
	// 每个如果不符合则当做string进行读取，并返回Token
	// ip="127.0.0.1" / ip=127.0.0.1
	// [ip,=,127.0.0.1]
	switch next {
	case `(`:
		// 直接返回Token 期望值: )、body、ip、icp、host、title、tls、string
		ret[TokenTypeLeftParenthesis] = "("
		sb.DeleteSpace()
		return ret, nil
	case `)`:
		// 直接返回Token 期望值: end、and、&&、or、||、body、ip、icp、host、title、tls、string
		ret[TokenTypeRightParenthesis] = ")"
		sb.DeleteSpace()
		return ret, nil
	case `=`:
		// 判断是 = 还是 == 返回Token 期望值: string
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "= expect other, not end"
			return ret, errors.New("= expect other, not end")
		}
		if n != "=" {
			sb.Reduce()
			ret[TokenTypeEquals] = "="
		} else {
			ret[TokenTypeStrongEquals] = "=="
		}
		sb.DeleteSpace()
		return ret, nil
	case `!`:
		// 判断是 !=，然后返回Token 期望值: string
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "! expect =, not end"
			return ret, errors.New("! expect =, not end")
		}
		if n != "=" {
			ret[TokenTypeError] = "! expect =, not " + n
			return ret, errors.New("! expect =, not " + n)
		}
		sb.DeleteSpace()
		ret[TokenTypeNotEquals] = "!="
		return ret, nil
	case `~`:
		// 判断是 ~= 还是 ~!= 返回Token 期望值: string
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "~ expect = or !=, not end"
			return ret, errors.New("~ expect = or !=, not end")
		}
		if n == "=" {
			sb.DeleteSpace()
			ret[TokenTypeRegexpEquals] = "~="
			return ret, nil
		}
		if n != "!" {
			ret[TokenTypeError] = "~ expect =/!, not " + n
			return ret, errors.New("~ expect =/!, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "~! expect =, not end"
			return ret, errors.New("~! expect =, not end")
		}
		if n != "=" {
			ret[TokenTypeError] = "~! expect =, not " + n
			return ret, errors.New("~! expect =, not " + n)
		}
		sb.DeleteSpace()
		ret[TokenTypeRegexpNotEquals] = "~!="
		return ret, nil
	case `a`:
		// 判断是 and 返回Token 期望值: (、body、ip、icp、host、title、tls、string
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "a expect nd, not end"
			return ret, errors.New("a expect nd, not end")
		}
		if n == "t" {
			n, e = sb.Next()
			if e {
				ret[TokenTypeError] = "at expect tr, not end"
				return ret, errors.New("at expect tr, not end")
			}
			if n != "t" {
				ret[TokenTypeError] = "at expect tr, not " + n
				return ret, errors.New("at expect tr, not " + n)
			}
			n, e = sb.Next()
			if e {
				ret[TokenTypeError] = "att expect r, not end"
				return ret, errors.New("att expect r, not end")
			}
			if n != "r" {
				ret[TokenTypeError] = "att expect r, not " + n
				return ret, errors.New("att expect r, not " + n)
			}
			ret[TokenTypeAttr] = "attr"
			sb.DeleteSpace()
			return ret, nil
		}
		if n != "n" {
			ret[TokenTypeError] = "a expect nd, not " + n
			return ret, errors.New("a expect nd, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "an expect d, not end"
			return ret, errors.New("an expect d, not end")
		}
		if n != "d" {
			ret[TokenTypeError] = "an expect d, not " + n
			return ret, errors.New("an expect d, not " + n)
		}
		// ret[TokenTypeAnd] = "and"
		ret[TokenTypeAND] = "&&"
		sb.DeleteSpace()
		return ret, nil
	case `&`:
		// 判断是 && 返回Token 期望值: (、body、ip、icp、host、title、tls、string
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "& expect &, not end"
			return ret, errors.New("& expect &, not end")
		}
		if n != "&" {
			ret[TokenTypeError] = "& expect &, not " + n
			return ret, errors.New("& expect &, not " + n)
		}
		sb.DeleteSpace()
		ret[TokenTypeAND] = "&&"
		return ret, nil
	case `o`:
		// 判断是 or 返回Token 期望值: (、body、ip、icp、host、title、tls、string
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "o expect r, not end"
			return ret, errors.New("o expect r, not end")
		}
		if n != "r" {
			ret[TokenTypeError] = "o expect r, not " + n
			return ret, errors.New("o expect r, not " + n)
		}
		//ret[TokenTypeOr] = "or"
		ret[TokenTypeOR] = "||"
		sb.DeleteSpace()
		return ret, nil
	case `|`:
		// 判断是 || 返回Token 期望值: (、body、ip、icp、host、title、tls、string
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "| expect |, not end"
			return ret, errors.New("| expect |, not end")
		}
		if n != "|" {
			ret[TokenTypeError] = "| expect |, not " + n
			return ret, errors.New("| expect |, not " + n)
		}
		sb.DeleteSpace()
		ret[TokenTypeOR] = "||"
		return ret, nil
	case `b`:
		// 判断是 body 返回Token 期望值: 可以包含空格 + =、==、!=、~=、~!=
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "b expect ody, not end"
			return ret, errors.New("b expect ody, not end")
		}
		if n != "o" {
			ret[TokenTypeError] = "b expect ody, not " + n
			return ret, errors.New("b expect ody, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "bo expect dy, not end"
			return ret, errors.New("bo expect dy, not end")
		}
		if n != "d" {
			ret[TokenTypeError] = "bo expect dy, not " + n
			return ret, errors.New("bo expect dy, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "bod expect y, not end"
			return ret, errors.New("bod expect y, not end")
		}
		if n != "y" {
			ret[TokenTypeError] = "bod expect y, not " + n
			return ret, errors.New("bod expect y, not " + n)
		}
		sb.DeleteSpace()
		ret[TokenTypeBody] = "body"
		return ret, nil
	case `i`:
		// 判断是 ip 还是icp 返回Token 期望值: 可以包含空格 + =、==、!=、~=、~!=
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "i expect p/cp, not end"
			return ret, errors.New("i expect p/cp, not end")
		}
		if n == "p" {
			sb.DeleteSpace()
			ret[TokenTypeIP] = "ip"
			return ret, nil // 判断是 ip 还是icp 返回Token 期望值: 可以包含空格 + =、==、!=、~=、~!=
		}
		if n != "c" {
			ret[TokenTypeError] = "i expect p/cp, not " + n
			return ret, errors.New("i expect p/cp, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "ic expect p, not end"
			return ret, errors.New("ic expect p, not end")
		}
		if n != "p" {
			ret[TokenTypeError] = "ic expect p, not " + n
			return ret, errors.New("ic expect p, not " + n)
		}
		sb.DeleteSpace()
		ret[TokenTypeICP] = "icp"
		return ret, nil
	case `h`:
		// 判断是host 返回Token 期望值: 可以包含空格 + =、==、!=、~=、~!=
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "h expect ost, not end"
			return ret, errors.New("h expect ost, not end")
		}
		if n != "o" {
			ret[TokenTypeError] = "h expect ost, not " + n
			return ret, errors.New("h expect ost, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "ho expect st, not end"
			return ret, errors.New("ho expect st, not end")
		}
		if n != "s" {
			ret[TokenTypeError] = "ho expect st, not " + n
			return ret, errors.New("ho expect st, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "hos expect t, not end"
			return ret, errors.New("hos expect t, not end")
		}
		if n != "t" {
			ret[TokenTypeError] = "hos expect t, not " + n
			return ret, errors.New("hos expect t, not " + n)
		}
		ret[TokenTypeHost] = "host"
		return ret, nil
	case `t`:
		// 判断是title 还是tls 返回Token 期望值: 可以包含空格 + =、==、!=、~=、~!=
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "t expect itle or ls, not end"
			return ret, errors.New("t expect itle or ls, not end")
		}
		if n != "i" && n != "l" {
			ret[TokenTypeError] = "t expect itle or ls, not n" + n
			return ret, errors.New("t expect itle or ls, not " + n)
		}
		if n == "i" {
			n, e = sb.Next()
			if e {
				ret[TokenTypeError] = "ti expect tle, not end"
				return ret, errors.New("ti expect tle, not end")
			}
			if n != "t" {
				ret[TokenTypeError] = "ti expect tle, not " + n
				return ret, errors.New("ti expect tle, not " + n)
			}
			n, e = sb.Next()
			if e {
				ret[TokenTypeError] = "tit expect le, not end"
				return ret, errors.New("tit expect le, not end")
			}
			if n != "l" {
				ret[TokenTypeError] = "tit expect le, not " + n
				return ret, errors.New("tit expect le, not " + n)
			}
			n, e = sb.Next()
			if e {
				ret[TokenTypeError] = "titl expect e, not end"
				return ret, errors.New("titl expect e, not end")
			}
			if n != "e" {
				ret[TokenTypeError] = "titl expect e, not " + n
				return ret, errors.New("titl expect e, not " + n)
			}
			sb.DeleteSpace()
			ret[TokenTypeTitle] = "title"
			return ret, nil
		} else {
			n, e = sb.Next()
			if e {
				ret[TokenTypeError] = "tl expect s, not end"
				return ret, errors.New("tl expect s, not end")
			}
			if n != "s" {
				ret[TokenTypeError] = "tl expect s, not " + n
				return ret, errors.New("tl expect s, not " + n)
			}
			sb.DeleteSpace()
			ret[TokenTypeTLS] = "tls"
			return ret, nil
		}
	case `p`:
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "p expect rotocol, not end"
			return ret, errors.New("p expect rotocol, not end")
		}
		if n != "r" {
			ret[TokenTypeError] = "p expect rotocol, not " + n
			return ret, errors.New("p expect rotocol " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "pr expect otocol, not end"
			return ret, errors.New("pr expect otocol, not end")
		}
		if n != "o" {
			ret[TokenTypeError] = "pr expect otocol, not " + n
			return ret, errors.New("pr expect otocol, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "pro expect tocol, not end"
			return ret, errors.New("pro expect tocol, not end")
		}
		if n != "t" {
			ret[TokenTypeError] = "pro expect tocol, not " + n
			return ret, errors.New("pro expect tocol, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "prot expect ocol, not end"
			return ret, errors.New("prot expect ocol, not end")
		}
		if n != "o" {
			ret[TokenTypeError] = "prot expect ocol, not " + n
			return ret, errors.New("prot expect ocol, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "proto expect col, not end"
			return ret, errors.New("proto expect col, not end")
		}
		if n != "c" {
			ret[TokenTypeError] = "proto expect col, not " + n
			return ret, errors.New("proto expect col, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "protoc expect ol, not end"
			return ret, errors.New("protoc expect ol, not end")
		}
		if n != "o" {
			ret[TokenTypeError] = "protoc expect ol, not " + n
			return ret, errors.New("protoc expect ol, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "protoco expect l, not end"
			return ret, errors.New("protoco expect l, not end")
		}
		if n != "l" {
			ret[TokenTypeError] = "protoco expect l, not " + n
			return ret, errors.New("protoco expect l, not " + n)
		}
		ret[TokenTypeProtocol] = "protocol"
		return ret, nil
	case `v`:
		n, e := sb.Next()
		if e {
			ret[TokenTypeError] = "v expect ersion, not end"
			return ret, errors.New("v expect ersion, not end")
		}
		if n != "e" {
			ret[TokenTypeError] = "v expect ersion, not " + n
			return ret, errors.New("v expect ersion " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "ve expect rsion, not end"
			return ret, errors.New("ve expect rsion, not end")
		}
		if n != "r" {
			ret[TokenTypeError] = "ve expect rsion, not " + n
			return ret, errors.New("ve expect rsion, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "ver expect sion, not end"
			return ret, errors.New("ver expect sion, not end")
		}
		if n != "s" {
			ret[TokenTypeError] = "ver expect sion, not " + n
			return ret, errors.New("ver expect sion, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "vers expect ion, not end"
			return ret, errors.New("vers expect ion, not end")
		}
		if n != "i" {
			ret[TokenTypeError] = "vers expect ion, not " + n
			return ret, errors.New("vers expect ion, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "versi expect on, not end"
			return ret, errors.New("versi expect on, not end")
		}
		if n != "o" {
			ret[TokenTypeError] = "versi expect on, not " + n
			return ret, errors.New("versi expect on, not " + n)
		}
		n, e = sb.Next()
		if e {
			ret[TokenTypeError] = "versio expect n, not end"
			return ret, errors.New("versio expect n, not end")
		}
		if n != "n" {
			ret[TokenTypeError] = "versio expect n, not " + n
			return ret, errors.New("versio expect n, not " + n)
		}
		ret[TokenTypeVersion] = "version"
		return ret, nil
	case `"`:
		// 开始读取字符串 直到遇到" 并且下一个字符为 空格、(、) 中的一个
		string_data := ""
		for {
			n, e := sb.Next()
			if e {
				if len(string_data) == 0 {
					ret[TokenTypeError] = "\" expect \", not end"
					return ret, errors.New("\" expect \", not end")
				} else if string(string_data[len(string_data)-1]) != "\"" {
					ret[TokenTypeError] = "\" expect \", not end"
					return ret, errors.New("\" expect \", not end")
				}
				break
			}
			if n == "\\" {
				n, e = sb.Next()
				if e {
					ret[TokenTypeError] = "\\ expect character to be escaped, not end"
					return ret, errors.New("\\ expect character to be escaped, not end")
				}
				string_data += n
				continue
			}
			if n == `"` {
				n, e = sb.Next()
				if e {
					break
				}
				if n == " " || n == "(" || n == ")" {
					sb.Reduce()
					break
				}
				ret[TokenTypeError] = "\" expect space or ( or ), not end"
				return ret, errors.New("\" expect space or ( or ), not end")
			}
			string_data += n
		}
		sb.DeleteSpace()
		ret[TokenTypeString] = string_data
		return ret, nil // 读取字符串 返回Token 期望值: (、)、空格
	case ` `:
		for {
			n, e := sb.Next()
			if e {
				ret[TokenTypeEnd] = ""
				return ret, nil
			}
			if n != " " {
				sb.Reduce()
				break
			}
		}
		ret[TokenTypeSpace] = " "
		return ret, nil // 读取空格，存在多个空格时一直读取，直到不是空格 返回Token 期望值: (、)、and、&&、or、||、body、ip、icp、host、title、tls、string
	default:
		ret[TokenTypeError] = "not parse, in " + next
		return ret, errors.New("not parse, in " + next)
	}
}
