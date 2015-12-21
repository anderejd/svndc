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

// TODO: append global svn options if provided
func svnCheckout(repos *url.URL, wcPath string, ga globalArgs) error {
	args := []string{ "checkout", repos.String(), wcPath }
	return execPiped("svn", args...)
}

// TODO: append global svn options if provided
func svnCommit(wcPath, message string, ga globalArgs) error {
	args := []string{"commit", wcPath, "--message", message}
	return execPiped("svn", args...)
}

func svnDiffCommit(ca commitArgs, ga globalArgs) (err error) {
	err = svnCheckout(ca.Repos, ca.WcPath, ga)
	if nil != err {
		return
	}
	err = cleanWcRoot(ca.WcPath)
	if nil != err {
		return
	}
	err = copyRecursive(ca.SrcPath, ca.WcPath)
	if nil != err {
		return
	}
	err = execPiped("svn", "add", ca.WcPath, "--force")
	if nil != err {
		return
	}
	out, err := execPrint("svn", "status", ca.WcPath)
	if nil != err {
		return
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		// svn remove all missing files
		fmt.Println(line)
	}
	return svnCommit(ca.WcPath, ca.Message, ga)
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
	ca := commitArgs{}
	ca.Message = "Hellooo :D"
	ca.WcPath = filepath.Join(testPath, "wc")
	ca.Repos, ca.SrcPath, err = setupTest(testPath)
	if nil != err {
		return
	}
	defer teardownTest(testPath)
	ga := globalArgs{}
	err = svnDiffCommit(ca, ga)
	if nil != err {
		return
	}
	fmt.Print("\n\nSelf test --> Success.\n\n\n")
	return nil
}

func parseArgs() (args cmdArgs, err error) {
	for i, arg := range os.Args {
		fmt.Println(i, ": ", arg)
	}
	return
}

type globalArgs struct {
	ConfigDir               *string // --config-dir ARG
	ConfigOption            *string // --config-options ARG
	NoAuthCache             bool    // --no-auth-cache
	NonInteractive          bool    // --non-ineractive
	Password                *string // --password ARG
	TrustServerCertFailures *string // --trust-server-cert-failures ARG 'unknown-ca' 'cn-mismatch' 'expired' 'not-yet-valid' 'other'
	Username                *string // --username ARG
}

type commitArgs struct {
	Message string   // --message ARG
	SrcPath string   // --src-path
	Repos   *url.URL // --dst-url
	WcPath  string   // --wc-path
}

type cmdArgs struct {
	Help        bool       // --help
	RunSelfTest bool       // --run-self-test
	CommitArgs  commitArgs
	GlobalArgs  globalArgs
}

func main() {
	args, err := parseArgs()
	if nil != err {
		log.Fatal(err)
	}
	fmt.Println(args)
	err = runSelfTest()
	if nil != err {
		log.Fatal(err)
	}
}
