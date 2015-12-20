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

struct testData {
	Path string;
	Content string;
	IsDir bool;
}

func makeTestData() []testData {
	var result []testData = {
		{ "1.txt", false, "data1" },
		{ "2.txt", false, "data2" },
		{ "subdir1", true, "" },
		{ filepath.Join("subdir1", "1.txt"), false, "subdata1" },
		{ "subdir2", true, "" }
	}
	return result
}

func createTestSourceFiles(basePath string) (err error) {
	err = os.Mkdir(path, 0755)
	if nil != err {
		return
	}
	testDatas := makeTestData(basePath)
	for _, td := range testDatas {
		if td.IsDir {
			err = os.Mkdir(td.Path)
			if nil != nil {
				return
			}
			continue
		}
		err = ioutil.WriteFile(td.Path, td.Content, 0755)
		if nil != nil {
			return
		}
	}
	return nil
}

func setupTest(testPath string) (repos *url.URL, srcPath string, err error) {
	err = os.Mkdir(testPath, 0755)
	if nil != err {
		return
	}
	srcPath = filepath.Join(testPath, "src")
	err = createTestSourceFiles(srcPath)
	if nil != err {
		return
	}
	reposPath := filepath.Join(testPath, "repos")
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
	testPath := filepath.Join(".", "self_test")
	reposUrl, srcPath, err := setupTest(testPath)
	if nil != err {
		return
	}
	defer teardownTest(testPath)
	wcPath := filepath.Join(testPath, "wc")
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
