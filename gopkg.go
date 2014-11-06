package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"errors"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	flag.Parse()

	switch flag.Arg(0) {
	case "":
		printHelp()
	case "get":
		getPackage()
	default:
		goCMD(append([]string{"go"}, flag.Args()...)...)
	}
}

func printHelp() {
	fmt.Println(`
  ________      __________ ____  __.________
 /  _____/  ____\______   \    |/ _/  _____/
/   \  ___ /  _ \|     ___/      </   \  ___
\    \_\  (  <_> )    |   |    |  \    \_\  \
 \______  /\____/|____|   |____|__ \______  /
        \/                        \/      \/ `)
	fmt.Println(`Usage:

	gopkg command [arguments]

The commands are:

    build       compile packages and dependencies
    clean       remove object files
    env         print Go environment information
    fix         run go tool fix on packages
    fmt         run gofmt on package sources
    get         download and install packages and dependencies
    install     compile and install packages and dependencies
    list        list packages
    run         compile and run Go program
    test        test packages
    tool        run specified go tool
    version     print Go version
    vet         run go tool vet on packages

Use "gopkg help [command]" for more information about a command.

Additional help topics:

    c           calling between Go and C
    filetype    file types
    gopath      GOPATH environment variable
    importpath  import path syntax
    packages    description of package lists
    testflag    description of testing flags
    testfunc    description of testing functions

Use "gopkg help [topic]" for more information about that topic.
`)
}

func goCMD(cmd ...string) {
	path, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	// 将packages添加到GOPATH变量
	os.Setenv("GOPATH", os.Getenv("GOPATH")+string(os.PathListSeparator)+toPath(path, "packages"))

	err = runCommand(cmd...)
	if err != nil {
		log.Fatal(err)
	}
}

func getPackage() {
	buf, err := ioutil.ReadFile("gopkg.yaml")
	if err != nil {
		log.Fatalln(err)
	}
	var p PKG
	err = yaml.Unmarshal(buf, &p)
	if err != nil {
		log.Fatalln(err)
	}

	goPaths := filepath.SplitList(os.Getenv("GOPATH"))
	// 获取第一个GOPATH位置
	goPath := goPaths[0]

	pkgPath := os.Getenv("GOOS") + "_" + os.Getenv("GOARCH")

	goGetCMD := append([]string{"go", "get"}, flag.Args()[1:]...)

	os.MkdirAll(toPath("packages", "src"), 0700)
	os.MkdirAll(toPath("packages", "pkg", pkgPath), 0700)

	for k, v := range p.Packages {
		fmt.Printf("%v:\n  - %v  ", k, v)

		// run go get
		err = runCommand(append(goGetCMD, v)...)
		if err != nil {
			fmt.Println("[ERROR]")
			log.Println(err)
			return
		}

		gocodeSrc := toPath(goPath, "src", v)
		packagesSrc := toPath("packages", "src", k)

		os.RemoveAll(packagesSrc)
		// Create symbolic link to packages/src
		err = os.Symlink(gocodeSrc, packagesSrc)
		if err != nil {
			fmt.Println("[ERROR]")
			log.Println(err)
			return
		}

		gocodePkg := toPath(goPath, "pkg", pkgPath, v)
		packagesPkg := toPath("packages", "pkg", pkgPath, k)

		// ".a" file or folder?
		_, err = os.Stat(gocodePkg + ".a")
		if err == nil {
			os.Remove(packagesPkg + ".a")
			err = os.Symlink(gocodePkg+".a", packagesPkg+".a")
			if err != nil {
				fmt.Println("[ERROR]")
				log.Println(err)
				return
			}
		} else {
			// Is a folder
			os.RemoveAll(packagesPkg)
			err = os.Symlink(gocodePkg, packagesPkg)
			if err != nil {
				fmt.Println("[ERROR]")
				log.Println(err)
				return
			}
		}
		fmt.Println("[OK]")
	}
}

func toPath(pathBuf ...string) (path string) {
	for _, v := range pathBuf {
		path += v + string(os.PathSeparator)
	}
	// packages/pkg/yaml.a/ ==> packages/pkg/yaml.a
	return path[:len(path)-1]
}

func runCommand(command ...string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Run()
	if err != nil {
		return err
	}

	bytesErr, err := ioutil.ReadAll(stderr)
	if err != nil {
		return err
	}

	if len(bytesErr) != 0 {
		return errors.New(string(bytesErr))
	}

	return nil
}

type PKG struct {
	Packages map[string]string `yaml:"packages"`
}
