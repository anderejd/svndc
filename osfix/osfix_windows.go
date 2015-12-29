package osfix

import "os"
import "os/exec"

// Workaround for the limitation in go 1.5 where os.RemoveAll does not remove
// all. In windows go 1.5 access denied is returned for read only files.
func RemoveAll(path string) error {
	fi, err := os.Stat(path)
	if nil != err {
		return err
	}
	if !fi.IsDir() {
		return os.Remove(path) // Change this to support readonly FILES
	}
	// Will remove readonly dirs with readonly files.
	return exec.Command("cmd.exe", "/C", "rmdir", "/S", "/Q", path).Run()
}
