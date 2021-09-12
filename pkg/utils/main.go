package utils

import (
	"regexp"
	"strings"
)

func ICPInfo(data string) string {
	icp_info := regexp.MustCompile("\\W[ ]{0,10}[I公]{1}[ ]{0,10}[C网]{1}[ ]{0,10}[P安]{1}[ ]{0,10}[证备][ ]{0,10}\\d+[ ]{0,10}[号\\-\\d ]{0,10}")
	icp_regexp := icp_info.FindAllString(data, -1)
	icp_data := ""
	icp_ := make(map[string]bool)
	for i := 0; i < len(icp_regexp); i++ {
		icp_regexp[i] = strings.ReplaceAll(icp_regexp[i], " ", "")
		if _, ok := icp_[icp_regexp[i]]; !ok {
			icp_[icp_regexp[i]] = true
			icp_data += icp_regexp[i] + "|"
		}
	}
	return strings.Trim(icp_data, "|")
}
