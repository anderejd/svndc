package main

import "fmt"
import "io/ioutil"
import "log"
import "net/url"
import "os"
import "os/exec"
import "path/filepath"
import "strings"


func cleanWcRoot(wc_path string) (err error) {
	infos, err := ioutil.ReadDir(wc_path)
	if nil != err {
		return
	}
	for _, inf := range infos {
		fmt.Println(inf.Name())
	}
	return nil
}


func execPiped(name string, arg ...string) error {
	fmt.Println(name + " " + strings.Join(arg, " "))
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin  = os.Stdin
	return cmd.Run()
}


func svnDiffCommit(repos *url.URL, wc_path string) (err error) {
	err = execPiped("svn", "checkout", repos.String(), wc_path)
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


func createRepos(repos_path string) (repos *url.URL, err error) {
	err = execPiped("svnadmin", "create", repos_path)
	if nil != err {
		return
	}
	abs_repos_path, err := filepath.Abs(repos_path)
	if nil != err {
		return
	}
	abs_repos_path = "file://" + abs_repos_path
	repos, err = url.Parse(abs_repos_path)
		return
}


func testSelf() (err error) {
	fmt.Print("\n\nSelf test --> Start...\n\n\n")
	test_path := "./self_test/"
	repos_path := test_path + "repos/"
	wc_path := test_path + "wc/"
	err = os.Mkdir(test_path, 0755)
	if nil != err {
		return
	}
	defer func() {
		inner_err := os.RemoveAll(test_path)
		if nil != inner_err {
			err = inner_err
		}
	}()
	repos_url, err := createRepos(repos_path)
	err = svnDiffCommit(repos_url, wc_path)
	if nil != err {
		return
	}
	fmt.Print("\n\nSelf test --> Success.\n\n\n")
	return nil
}


func main() {
	err := testSelf()
	if nil != err {
		log.Fatal(err)
	}
}


