/*
* @Author: detailyang
* @Date:   2016-02-09 17:39:51
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-09 21:17:06
 */

package utils

import (
	"errors"
)

var (
	ErrHttpRedirect    = errors.New("get redirect")
	ErrEmptyDialerPool = errors.New("get empty dialer pool")
)
