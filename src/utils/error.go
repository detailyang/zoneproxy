/*
* @Author: detailyang
* @Date:   2016-02-09 17:39:51
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-15 12:24:43
 */

package utils

import (
	"errors"
)

var (
	ErrHttpRedirect    = errors.New("get redirect")
	ErrEmptyDialerPool = errors.New("get empty dialer pool")
	ErrReadConfig      = errors.New("read error config")
)
