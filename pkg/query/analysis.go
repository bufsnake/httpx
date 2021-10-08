package query

import (
	"encoding/json"
	"strconv"
	"strings"
)

// 返回: sql预编译语句、参数列表、query格式化、error
func AnalysisQuery(input string) (sql string, params []interface{}, queryFormat string, err error) {
	input = strings.Trim(input, " ")
	tokens := make([]map[string]string, 0)
	tokens = append(tokens, map[string]string{TokenTypeStart: "start"})
	buffer := strbuffer{input: []rune(input), index: 0}
	for {
		token, err := buffer.LexicalAnalysis()
		if err != nil {
			return "", nil, "", err
		}
		if _, ok := token[TokenTypeEnd]; ok {
			break
		}
		tokens = append(tokens, token)
	}
	tokens = append(tokens, map[string]string{TokenTypeEnd: "end"})
	tb := NewTokenBuffer(tokens)
	err = tb.GrammaAnalysis()
	if err != nil {
		return "", nil, "", err
	}
	// 格式化输入
	formatStr := ""
	for index, token := range tokens {
		for t, v := range token {
			switch t {
			case TokenTypeLeftParenthesis, TokenTypeEquals, TokenTypeStrongEquals, TokenTypeNotEquals, TokenTypeRegexpEquals, TokenTypeRegexpNotEquals, TokenTypeBody, TokenTypeIP, TokenTypeProtocol, TokenTypeVersion, TokenTypeAttr, TokenTypeICP, TokenTypeHost, TokenTypeTitle, TokenTypeTLS:
				formatStr += v
			case TokenTypeRightParenthesis:
				// 只有写一个Token为)的时候不加空格
				for tt, _ := range tokens[index+1] {
					switch tt {
					case TokenTypeRightParenthesis:
						formatStr += v
					default:
						formatStr += v + " "
					}
				}
			case TokenTypeAND, TokenTypeOR:
				formatStr += v + " "
			case TokenTypeString:
				tts := make([]byte, 0)
				// JSON 对部分字符进行Unicode编码
				tts, err = json.Marshal(v)
				if err != nil {
					return "", nil, "", err
				}
				temp := string(tts) + " "
				nextToken := tokens[index+1]
				for n, _ := range nextToken {
					if n == TokenTypeRightParenthesis {
						temp = string(tts)
					}
				}
				str, err := strconv.Unquote(strings.Replace(strconv.Quote(temp), `\\u`, `\u`, -1))
				if err != nil {
					return "", nil, "", err
				}
				formatStr += str
				break
			case TokenTypeStart, TokenTypeEnd:
				break
			default:
				formatStr += v + " "
			}
		}
	}
	// 获取SQL语句以及输入
	// 特殊点 字符串->如果前面一个字符不为 = ,则是全局搜索，需要对所有关键字进行 = 搜索,最后合并之后再两边加()
	sqlStr := ""
	paramArray := make([]interface{}, 0)
	for index, token := range tokens {
		for t, v := range token {
			// TokenTypeEquals
			// TokenTypeStrongEquals
			// ...
			// 以上判断下个Token是否为字符串，且字符串的长度是否为空
			switch t {
			case TokenTypeLeftParenthesis:
				sqlStr += v
				break
			case TokenTypeRightParenthesis:
				// 只有写一个Token为)的时候不加空格
				for tt, _ := range tokens[index+1] {
					switch tt {
					case TokenTypeRightParenthesis:
						sqlStr += v
					default:
						sqlStr += v + " "
					}
				}
				break
			case TokenTypeBody, TokenTypeIP, TokenTypeProtocol, TokenTypeVersion, TokenTypeAttr, TokenTypeICP, TokenTypeHost, TokenTypeTitle, TokenTypeTLS:
				sqlStr += "`" + v + "` "
				break
			case TokenTypeEquals:
				sqlStr += "LIKE ?"
				break
			case TokenTypeStrongEquals:
				sqlStr += "= ?"
				break
			case TokenTypeNotEquals:
				// 判断后一个字符是否为空
				for _, va := range tokens[index+1] {
					if va == "" {
						sqlStr += "<> ?"
					} else {
						sqlStr += "NOT LIKE ?"
					}
				}
				break
			case TokenTypeRegexpEquals:
				sqlStr += "REGEXP ?"
				break
			case TokenTypeRegexpNotEquals:
				sqlStr += "NOT REGEXP ?"
				break
			case TokenTypeAND:
				sqlStr += "AND "
				break
			case TokenTypeOR:
				sqlStr += "OR "
				break
			case TokenTypeString:
				temp_sql := ""
				need := true
				for tt, _ := range tokens[index-1] {
					// 判断字符串的前一个Token，为特定的Token添加特定的值
					switch tt {
					case TokenTypeLeftParenthesis, TokenTypeStart, TokenTypeAND, TokenTypeOR:
						temp_str := []string{"%" + v + "%", "%" + v + "%", "%" + v + "%", "%" + v + "%", "%" + v + "%", "%" + v + "%"}
						for ts := 0; ts < len(temp_str); ts++ {
							paramArray = append(paramArray, temp_str[ts])
						}
						break
					case TokenTypeEquals:
						paramArray = append(paramArray, "%"+v+"%")
						break
					case TokenTypeNotEquals:
						if v == "" {
							paramArray = append(paramArray, v)
						} else {
							paramArray = append(paramArray, "%"+v+"%")
						}
						break
					case TokenTypeStrongEquals, TokenTypeRegexpEquals, TokenTypeRegexpNotEquals:
						paramArray = append(paramArray, v)
						break
					}
					if strings.Contains(tt, "Equals") {
						need = false
						break
					}
				}
				if need {
					temp_sql = "(`body` LIKE ? OR `ip` LIKE ? OR `icp` LIKE ? OR `host` LIKE ? OR `title` LIKE ? OR `tls` LIKE ?)"
				}
				for tt, _ := range tokens[index+1] {
					switch tt {
					case TokenTypeRightParenthesis:
						sqlStr += temp_sql
					default:
						sqlStr += temp_sql + " "
					}
				}
				break
			case TokenTypeStart, TokenTypeEnd:
				break
			}
		}
	}
	return strings.Trim(sqlStr, " "), paramArray, strings.Trim(formatStr, " "), nil
}
