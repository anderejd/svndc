// +build !windows

package osfix

import "os"

func RemoveAll(path string) error {
	return os.RemoveAll(path)
}
