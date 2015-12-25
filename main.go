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
import "reflect"
import "strings"

const help = `--help           Print syntax help
--run-self-test  Requires svnadmin. Will create a local repository in 
                 the directory ./self_test/repos and use for tests. The
                 directory ./self will be deleted when tests complete.
--src-path       Path to directory with files to commit
--dst-url        Target SVN repository URL (commit destination)
--wc-path        Working copy path. This path will be created by svn
                 checkout, if it does not exist. Files from --src-path 
                 will be copied here. Files not present in --src-path
                 will be svn-deleted in --wc-path.
--wc-delete      Will delete --wc-path after svn commit.
--message        Message for svn commit.

SVN Global args (see svn documentaion):

--config-dir ARG
--config-options ARG
--no-auth-cache
--non-ineractive
--password ARG
--trust-server-cert-failures ARG
--username ARG
`

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
func svnCheckout(repos url.URL, wcPath string, ga globalArgs) error {
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
			err = errors.New("Unknown status line: " + line)
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
	repos, err := url.Parse(ca.Repos)
	if nil != err {
		return
	}
	err = svnCheckout(*repos, ca.WcPath, ga)
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

func createRepos(reposPath string) (reposUrl string, err error) {
	err = execPiped("svnadmin", "create", reposPath)
	if nil != err {
		return
	}
	absReposPath, err := filepath.Abs(reposPath)
	if nil != err {
		return
	}
	absReposPath = "file://" + absReposPath
	repos, err := url.Parse(absReposPath)
	if nil != err {
		return
	}
	reposUrl = repos.String()
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

func setupTest(testPath string) (reposUrl string, srcPath string, err error) {
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
	reposUrl, err = createRepos(reposPath)
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

type argMap map[string]*string

// Allows keys with nil values.
// Disallow multiple values for a single key.
func getArgMap(args []string) (am argMap, err error) {
	key := ""
	am = argMap{}
	for _, arg := range args {
		clean := strings.TrimSpace(arg)
		if strings.HasPrefix(clean, "--") {
			key = clean
			_, hasKey := am[key]
			if !hasKey {
				am[key] = nil
				continue
			}
			err = errors.New("Duplicate keys: " + key)
			return
		}
		if "" == key {
			err = errors.New("Expected key (--), found: " + clean)
			return
		}
		v := am[key]
		if nil != v {
			err = errors.New("Expected single value for: " + key)
		}
		am[key] = &clean // TODO: Investigate, why does this work? Pointer escape analysis?
	}
	return
}

// TODO: Figure out how to support more types, specifically url.URL in a clean way
func parseArgs(src []string, out interface{}) (err error) {
	am, err := getArgMap(src[1:])
	if nil != err {
		return
	}
	fm, err := buildFieldMap(reflect.TypeOf(out).Elem(), "cmd")
	if nil != err {
		return
	}
	rv := reflect.ValueOf(out).Elem()
	for k, v := range am {
		fieldIndex, hasKey := fm[k]
		if !hasKey {
			err = errors.New("Unknown option: " + k)
			return
		}
		field := rv.FieldByIndex(fieldIndex)
		if reflect.Bool == field.Type().Kind() {
			if nil != v {
				err = errors.New("Syntax error: " + k + " " + *v)
				return
			}
			rv.FieldByIndex(fieldIndex).SetBool(true)
			continue
		}
		if nil == v {
			err = errors.New("Value missing for key: " + k)
		}
		fmt.Println(k, ", ", *v)
		field.Set(reflect.ValueOf(*v))
	}
	return
}

type fieldIndex []int
type fieldMap map[string]fieldIndex

func (m fieldMap) Append(t reflect.Type, index []int, tagName string) error {
	numf := t.NumField()
	for i := 0; i < numf; i++ {
		field := t.Field(i)
		fieldIndex := append(index, i)
		if field.Type.Kind() == reflect.Struct && field.Anonymous { // only traverse into embedded structs
			err := m.Append(field.Type, fieldIndex, tagName)
			if nil != err {
				return err
			}
			continue
		}
		name, err := getFieldName(field, tagName)
		if nil != err {
			return err
		}
		m[name] = fieldIndex
	}
	return nil
}

func getFieldName(sf reflect.StructField, tagName string) (name string, err error) {
	name = sf.Tag.Get(tagName)
	if "" == name {
		sfstr := fmt.Sprintf("%#v", sf)
		err = errors.New("getArgName failed for: " + sfstr)
		return
	}
	return
}

func buildFieldMap(t reflect.Type, tagName string) (fm fieldMap, err error) {
	fm = fieldMap{}
	index := []int{}
	err = fm.Append(t, index, tagName)
	return fm, err
}

type globalArgs struct {
	ConfigDir               string `cmd:"--config-dir"`
	ConfigOption            string `cmd:"--config-options"`
	NoAuthCache             bool   `cmd:"--no-auth-cache"`
	NonInteractive          bool   `cmd:"--non-ineractive"`
	Password                string `cmd:"--password"`
	TrustServerCertFailures string `cmd:"--trust-server-cert-failures"`
	Username                string `cmd:"--username"`
}

type commitArgs struct {
	Message  string `cmd:"--message"`
	SrcPath  string `cmd:"--src-path"`
	Repos    string `cmd:"--dst-url"`
	WcPath   string `cmd:"--wc-path"`
	WcDelete bool   `cmd:"--wc-delete"`
}

type cmdArgs struct {
	Help        bool `cmd:"--help"`
	RunSelfTest bool `cmd:"--run-self-test"`
	commitArgs
	globalArgs
}

func printUsage() {
	fmt.Println(help)
}

func parseOsArgs() (args cmdArgs, err error) {
	if len(os.Args) < 2 {
		args.Help = true
		return
	}
	err = parseArgs(os.Args, &args)
	return
}

func main() {
	args, err := parseOsArgs()
	if nil != err {
		printUsage()
		log.Fatal(err)
	}
	if args.Help {
		printUsage()
		return
	}
	if args.RunSelfTest {
		err = runSelfTest()
	}
	if nil != err {
		log.Fatal(err)
	}
}
