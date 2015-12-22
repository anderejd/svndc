package main

import "errors"
import "fmt"
import "io"
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
	args := []string{"checkout", repos.String(), wcPath}
	return execPiped("svn", args...)
}

// TODO: append global svn options if provided
func svnCommit(wcPath, message string, ga globalArgs) error {
	args := []string{"commit", wcPath, "--message", message}
	return execPiped("svn", args...)
}

func svnGetMissing(wcPath string) (missing []string, err error) {
	out, err := execPrint("svn", "status", wcPath)
	if nil != err {
		return
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 3 {
			continue
		}
		if line[0] != '!' {
			continue
		}
		if ' ' != line[1] && '\t' != line[1] {
			err = errors.New("Unkown status line: " + line)
			return
		}
		p := strings.TrimSpace(line[1:])
		missing = append(missing, p)
	}
	return
}

func svnDeleteMissing(wcPath string) (err error) {
	missing, err := svnGetMissing(wcPath)
	if nil != err {
		return
	}
	if len(missing) == 0 {
		return
	}
	args := append([]string{"rm"}, missing...)
	err = execPiped("svn", args...)
	if nil != err {
		return
	}
	return
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
	err = svnDeleteMissing(ca.WcPath)
	if nil != err {
		return
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
		{"subdir_a", true, ""},
		{filepath.Join("subdir_a", "3.txt"), false, "data3"},
		{"subdir_b", true, ""},
		{filepath.Join("subdir_b", "4.txt"), false, "data4"},
		{"subdir_c", true, ""}}
	return result
}

func removeSomeTestFiles(srcPath string) (err error) {
	err = os.Remove(filepath.Join(srcPath, "1.txt"))
	if nil != err {
		return
	}
	err = os.Remove(filepath.Join(srcPath, "subdir_a", "3.txt"))
	if nil != err {
		return
	}
	return os.RemoveAll(filepath.Join(srcPath, "subdir_b"))
}

const perm = 0755

func crateTestFiles(basePath string, tds []testData) (err error) {
	err = os.Mkdir(basePath, perm)
	if nil != err {
		return
	}
	for _, td := range tds {
		err = createTestFile(td, basePath)
		if nil != err {
			return
		}
	}
	return nil
}

func createTestFile(td testData, basePath string) error {
	path := filepath.Join(basePath, td.Path)
	if td.IsDir {
		return os.Mkdir(path, perm)
	}
	return ioutil.WriteFile(path, []byte(td.Content), perm)
}

func setupTest(testPath string) (repos *url.URL, srcPath string, err error) {
	err = os.Mkdir(testPath, perm)
	if nil != err {
		return
	}
	srcPath = filepath.Join(testPath, "src")
	tds := makeTestData()
	err = crateTestFiles(srcPath, tds)
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
	err = removeSomeTestFiles(ca.SrcPath)
	if nil != err {
		return
	}
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
	args.RunSelfTest = true
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
	Message  string   // --message ARG
	SrcPath  string   // --src-path ARG
	Repos    *url.URL // --dst-url ARG
	WcPath   string   // --wc-path ARG
	WcDelete bool     // --wc-delete
}

type cmdArgs struct {
	Help        bool // --help
	RunSelfTest bool // --run-self-test
	CommitArgs  commitArgs
	GlobalArgs  globalArgs
}

func main() {
	args, err := parseArgs()
	if nil != err {
		log.Fatal(err)
	}
	if args.RunSelfTest {
		err = runSelfTest()
	} else if args.Help {
		// print usage
	}
	if nil != err {
		log.Fatal(err)
	}
}
