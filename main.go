package main

import "fmt"
import "io/ioutil"
import "log"
import "os"
import "os/exec"


func execPiped(name string, arg ...string) error {
	cmd := exec.Command(name, arg)
	cmd.Stdout = os.Stdout
	cmd.Stdin  = os.Stdin
	return cmd.Run()
}


func cleanWcRoot(wc_path string) err error {
	infos, err := ioutil.ReadDir(wc_path)
	if nil != err {
		return
	}
	for _, inf := range infos {
		fmt.Println(inf)
	}
	return nil
}


func svnDiffCommit(repos_path string, wc_path string) err error{
	err = execPiped("svn", "checkout", repos_path, wc_path)
	if nil != err {
		return
	}
	err = cleanWcRoot(wc_path)
	if nil != err {
		return
	}
	// copy all files and folders from source to working copy
	// svn status
	// svn remove all missing files
	// svn commit
	return nil
}


func testSelf() err error {
	fmt.Println("Self test --> Start...")
	repos_path := "./self_test/repos/"
	wc_path := "./self_test/wc/"
	err = execPiped("svnadmin", "create", repos_path)
	if nil != err {
		return
	}
	err = svnDiffCommit(repos_path, wc_path)
	if nil != err {
		return
	}
	err = os.RemoveAll(repos_path)
	if nil != err {
		return
	}
	fmt.Println("Self test --> Done.")
}


func main() {
	err := testSelf()
	if nil != err {
		log.Fatal(err)
	}
}


