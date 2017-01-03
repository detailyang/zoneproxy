/*
* @Author: detailyang
* @Date:   2016-02-09 02:36:31
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-15 10:57:05
 */

package utils

import (
	"regexp"
)

type ACL struct {
}

func (self *ACL) MatchHost(patterns []string, host string) bool {
	for _, pattern := range patterns {
		rp, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		ok := rp.MatchString(host)
		if ok == false {
			continue
		}
		return true
	}

	return false
}
