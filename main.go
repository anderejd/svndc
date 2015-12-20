package main

import "fmt"
import "io/ioutil"
import "io"
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

func copyFile(src, dst string) (err error) {
	s, err := os.Open(src)
	if err != nil {
		return
	}
	defer func() {
		closeErr := s.Close()
		if nil == err {
			err = closeErr
		}
	}()
	d, err := os.Create(dst)
	if err != nil {
		return
	}
	_, err = io.Copy(d, s)
	if nil != err {
		d.Close()
		return
	}
	return d.Close()
}

func copyRecursive(srcDir, dstDir string) (err error) {
	err = os.MkdirAll(dstDir, perm)
	if nil != err {
		return
	}
	infs, err := ioutil.ReadDir(srcDir)
	if nil != err {
		return
	}
	for _, inf := range infs {
		src := filepath.Join(srcDir, inf.Name())
		dst := filepath.Join(dstDir, inf.Name())
		if inf.IsDir() {
			err = copyRecursive(src, dst)
			if nil != err {
				return
			}
			continue
		}
		err = copyFile(src, dst)
		if nil != err {
			return
		}
	}
	return nil
}

func execPrint(name string, arg ...string) ([]byte, error) {
	fmt.Println(name + " " + strings.Join(arg, " "))
	return exec.Command(name, arg...).Output()
}

type svnOptions struct {
	Username    *string // --username ARG
	Password    *string // --password ARG
	NoAuthCache bool    // --no-auth-cache
	//REQUIRED NonInteractive bool // --non-ineractive
	TrustServerCertFailures *string // --trust-server-cert-failures ARG 'unknown-ca' 'cn-mismatch' 'expired' 'not-yet-valid' 'other'
	ConfigDir               *string // config-dir ARG
	ConfigOption            *string // --config-options ARG
	CommitMessage           string  // --message ARG
}

func makeGlobalArgs(opts svnOptions) []string {
	return []string{}
}

func svnDiffCommit(
	srcPath string,
	wcPath string,
	repos *url.URL,
	opts svnOptions) (err error) {
	err = execPiped("svn", "checkout", repos.String(), wcPath)
	if nil != err {
		return
	}
	err = cleanWcRoot(wcPath)
	if nil != err {
		return
	}
	err = copyRecursive(srcPath, wcPath)
	if nil != err {
		return
	}
	err = execPiped("svn", "add", wcPath, "--force")
	if nil != err {
		return
	}
	out, err := execPrint("svn", "status", wcPath)
	if nil != err {
		return
	}
	statusLines := strings.Split(string(out), "\n")
	for _, line := range statusLines {
		// svn remove all missing files
		fmt.Println(line) // TODO: avoid printing an extra line
	}
	commitArgs := []string{"commit", wcPath}
	commitArgs = append(commitArgs, "--message", "hej hej :D")
	return execPiped("svn", commitArgs...)
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

type testData struct {
	Path    string
	IsDir   bool
	Content string
}

func makeTestData() []testData {
	result := []testData{
		{"1.txt", false, "data1"},
		{"2.txt", false, "data2"},
		{"subdir1", true, ""},
		{filepath.Join("subdir1", "1.txt"), false, "subdata1"},
		{"subdir2", true, ""}}
	return result
}

const perm = 0755

func createTestSourceFiles(basePath string) (err error) {
	err = os.Mkdir(basePath, perm)
	if nil != err {
		return
	}
	testDatas := makeTestData()
	for _, td := range testDatas {
		path := filepath.Join(basePath, td.Path)
		if td.IsDir {
			err = os.Mkdir(path, perm)
			if nil != err {
				return
			}
			continue
		}
		err = ioutil.WriteFile(path, []byte(td.Content), perm)
		if nil != err {
			return
		}
	}
	return nil
}

func setupTest(testPath string) (repos *url.URL, srcPath string, err error) {
	err = os.Mkdir(testPath, perm)
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
	var opts svnOptions
	err = svnDiffCommit(srcPath, wcPath, reposUrl, opts)
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
