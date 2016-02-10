/*
* @Author: detailyang
* @Date:   2016-02-09 02:36:31
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-10 19:16:06
 */

package utils

import (
	"regexp"
)

type ACL struct {
}

func (self *ACL) MatchHost(pattern, host string) bool {
	rp, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return rp.MatchString(host)
}
