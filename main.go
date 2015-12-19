package main

import "fmt"
import "io/ioutil"
import "log"
import "net/url"
import "os"
import "os/exec"
import "path/filepath"
import "strings"

func cleanWcRoot(wcPath string) (err error) {
	infos, err := ioutil.ReadDir(wcPath)
	if nil != err {
		return
	}
	for _, inf := range infos {
		if ".svn" == inf.Name() {
			continue
		}
		fullPath := filepath.Join(wcPath, inf.Name())
		err = os.RemoveAll(fullPath)
		if nil != err {
			return
		}
	}
	return nil
}

func execPiped(name string, arg ...string) error {
	fmt.Println(name + " " + strings.Join(arg, " "))
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func svnDiffCommit(srcPath string, wcPath string, repos *url.URL) (err error) {
	err = execPiped("svn", "checkout", repos.String(), wcPath)
	if nil != err {
		return
	}
	err = cleanWcRoot(wcPath)
	if nil != err {
		return
	}
	fmt.Println(srcPath) // remove later
	// copy all files and folders from source to working copy
	// svn status
	// svn remove all missing files
	// svn commit
	return nil
}

func createRepos(reposPath string) (repos *url.URL, err error) {
	err = execPiped("svnadmin", "create", reposPath)
	if nil != err {
		return
	}
	absReposPath, err := filepath.Abs(reposPath)
	if nil != err {
		return
	}
	absReposPath = "file://" + absReposPath
	repos, err = url.Parse(absReposPath)
	return
}

func createTestSourceFiles(path string) (err error) {
	err = os.Mkdir(path, 0755)
	if nil != err {
		return
	}
	// create test files and folders
	return nil
}

func setupTest(testPath string) (repos *url.URL, srcPath string, err error) {
	err = os.Mkdir(testPath, 0755)
	if nil != err {
		return
	}
	srcPath = testPath + "src/"
	err = createTestSourceFiles(srcPath)
	if nil != err {
		return
	}
	reposPath := testPath + "repos/"
	repos, err = createRepos(reposPath)
	return
}

func teardownTest(testPath string) {
	err := os.RemoveAll(testPath)
	if nil != err {
		log.Println("ERROR: ", err)
	}
}

func runSelfTest() (err error) {
	fmt.Print("\n\nSelf test --> Start...\n\n\n")
	testPath := "./self_test/"
	reposUrl, srcPath, err := setupTest(testPath)
	if nil != err {
		return
	}
	fmt.Println(srcPath) // remove later
	defer teardownTest(testPath)
	wcPath := testPath + "wc/"
	err = svnDiffCommit(srcPath, wcPath, reposUrl)
	if nil != err {
		return
	}
	fmt.Print("\n\nSelf test --> Success.\n\n\n")
	return nil
}

func main() {
	err := runSelfTest()
	if nil != err {
		log.Fatal(err)
	}
}
