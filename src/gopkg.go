/*
 Copyright 2015 Bluek404

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"packages/yaml"
)

const (
	version = "0.0.1"

	filePerm os.FileMode = 0644 // -rw-r--r--
	dirPerm  os.FileMode = 0755 // drwxr-xr-x
)

func printHelp() {
	fmt.Println(`
  ________      __________ ____  __.________
 /  _____/  ____\______   \    |/ _/  _____/
/   \  ___ /  _ \|     ___/      </   \  ___
\    \_\  (  <_> )    |   |    |  \    \_\  \
 \______  /\____/|____|   |____|__ \______  /
        \/                        \/      \/`, "v"+version)
	fmt.Println(`Usage:

	gopkg command [arguments]

The commands are:

	new         create a new package
	test        test packages
	run         compile and run Go program
	build       compile packages and dependencies

Use "gopkg [command] -h" for more information about a command.
`)
}

func toPath(p ...string) (path string) {
	for _, v := range p {
		path += v + string(os.PathSeparator)
	}
	// packages/pkg/yaml.a/ ==> packages/pkg/yaml.a
	return path[:len(path)-1]
}

func newPackage(name string, lib bool) {
	os.MkdirAll(toPath(name, "src"), dirPerm)
	if lib {
		data := "package " + name + "\n" +
			"\n" +
			"func " + name + "() string {\n" +
			"	return \"Hello World!\"\n" +
			"}"
		ioutil.WriteFile(toPath(name, "src", "lib.go"), []byte(data), filePerm)
	} else {
		data := "package main\n" +
			"\n" +
			"import (\n" +
			"	\"fmt\"\n" +
			")\n" +
			"\n" +
			"func main() {\n" +
			"	fmt.Println(\"Hello World!\")\n" +
			"}"
		ioutil.WriteFile(toPath(name, "src", "main.go"), []byte(data), filePerm)
	}
	gopkgCfg := "name: " + name + "\n" +
		"authors:\n" +
		"  - Your Name <email@example.com>\n" +
		"\n" +
		"packages:\n" +
		"  - name: package\n" +
		"    git: https://github.com/example/package\n" +
		"    rev: 49c95bdc21843256fb6c4e0d370a05f24a0bf213"
	ioutil.WriteFile(toPath(name, "gopkg.yaml"), []byte(gopkgCfg), filePerm)
}

func getArg(i int) string {
	if len(os.Args) > i {
		return os.Args[i]
	} else {
		return ""
	}
}

func runCommand(command ...string) error {
	return runCommandInDir("", command...)
}

func runCommandInDir(dir string, command ...string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	} else {
		return fi.IsDir()
	}
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	} else {
		return !fi.IsDir()
	}
}

func copyFile(src, dst string) (int64, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	return io.Copy(dstFile, srcFile)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		// 得到目标路径
		dstPath := strings.Replace(path, src, "", 1)
		dstPath = toPath(dst, dstPath)
		if f.IsDir() {
			return os.MkdirAll(dstPath, dirPerm)
		} else {
			_, err := copyFile(path, dstPath)
			return err
		}
	})
}

func greenText(s string) string {
	return "\033[32;1m" + s + "\033[0m"
}

func yellowText(s string) string {
	return "\033[33;1m" + s + "\033[0m"
}

type gopkgCfg struct {
	Name     string   `yaml:"name"`
	Authors  []string `yaml:"authors"`
	Packages []struct {
		Name   string `yaml:"name"`
		Git    string `yaml:"git"`
		Rev    string `yaml:"rev"`
		Tag    string `yaml:"tag"`
		Branch string `yaml:"branch"`
	} `yaml:"packages"`
}

func isSrcFile(fileName string) bool {
	fileNameLen := len(fileName)
	srcFileList := []string{".go", ".c", ".h", ".s", ".cpp"}
	for _, srcFile := range srcFileList {
		srcFileLen := len(srcFile)
		if fileNameLen > srcFileLen {
			if fileName[fileNameLen-srcFileLen:] == srcFile {
				return true
			}
		}
	}
	return false
}

func haveSrcFiles(path string) bool {
	var haveSrcFilesErr = errors.New("YES")
	err := filepath.Walk(path, func(p string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if isSrcFile(f.Name()) {
			return haveSrcFilesErr
		}

		return nil
	})
	if err == haveSrcFilesErr {
		return true
	}
	return false
}

// 将普通 package 转换为 gopkg 的格式
func conToGopkg(path string) error {
	srcPath := toPath(path, "src")
	err := os.Mkdir(srcPath, dirPerm)
	if err != nil {
		return err
	}
	ignoreDir := []string{toPath(path, ".git"), srcPath}

	// move all source files to ./src
	err = filepath.Walk(path, func(p string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			if haveSrcFiles(path) {
				return os.MkdirAll(p, dirPerm)
			}
			ignoreDir = append(ignoreDir, p)
			return nil
		}

		for _, dirName := range ignoreDir {
			if p[:len(dirName)] == dirName {
				return nil
			}
		}

		file := strings.Replace(p, path, "", 1)
		if isSrcFile(file) {
			_, err = copyFile(p, toPath(srcPath, file))
			if err != nil {
				return err
			}
			return os.Remove(p)
		}

		return nil
	})

	// TODO: 自动分析 package 依赖
	// TODO: 自动将 package 转换为 gopkg 格式并生成 gopkg.yaml
	return nil
}

func getDeps(path string) (*gopkgCfg, error) {
	buf, err := ioutil.ReadFile(toPath(path, "gopkg.yaml"))
	if err != nil {
		return nil, err
	}
	var p = new(gopkgCfg)
	err = yaml.Unmarshal(buf, p)
	if err != nil {
		return nil, err
	}

	tempDir := toPath(os.TempDir(), "gopkg-"+randomStr())
	//defer os.RemoveAll(tempDir)
	for _, pkg := range p.Packages {
		pkgDir := toPath("src", "packages", pkg.Name)
		if dirExists(pkgDir) {
			continue
		} else {
			fmt.Println(greenText("Getting"), pkg.Name, "["+pkg.Git+"]")

			gitPath := toPath(tempDir, pkg.Name)
			err := runCommand("git", "clone", "-q", pkg.Git, gitPath)
			if err != nil {
				os.Exit(1)
			}
			if pkg.Branch != "" {
				fmt.Println("  - Branch:", pkg.Branch)
				err = runCommandInDir(gitPath, "git", "checkout", "-q", pkg.Branch)
				if err != nil {
					os.Exit(1)
				}
			}
			if pkg.Tag != "" {
				fmt.Println("  - Tag:", pkg.Tag)
				err = runCommandInDir(gitPath, "git", "checkout", "-q", pkg.Tag)
				if err != nil {
					os.Exit(1)
				}
			}
			if pkg.Rev != "" {
				fmt.Println("  - Rev:", pkg.Rev)
				err = runCommandInDir(gitPath, "git", "reset", "-q", "--hard", pkg.Rev)
				if err != nil {
					os.Exit(1)
				}
			}

			if !fileExists(toPath(gitPath, "gopkg.yaml")) {
				fmt.Println("  - [" + yellowText("Not used GoPKG") + "]\n")

				err = conToGopkg(gitPath)
				if err != nil {
					return nil, err
				}
			}

			pkgPath := toPath("src", "packages", pkg.Name)
			// 将 src 内源码移到 packages 目录中
			err = copyDir(toPath(gitPath, "src"), pkgPath)
			if err != nil {
				return nil, err
			}
			// 将剩余其他文件移到 packages 目录中（README、LICENSE等等）
			err = filepath.Walk(gitPath, func(path string, f os.FileInfo, err error) error {
				if f == nil {
					return err
				}
				if f.IsDir() {
					return nil
				}
				_, err = copyFile(path, toPath(pkgPath, f.Name()))
				return err
			})
			if err != nil {
				return nil, err
			}

			// move ./src/packages/xxxx/packages to
			// ./src/packages
			pkgPkgPath := toPath(pkgPath, "packages")
			copyDir(pkgPkgPath, toPath("src", "packages"))
			os.RemoveAll(pkgPkgPath)

			fmt.Println("  - " + greenText("Done") + "\n")

			_, err = getDeps(gitPath)
			if err != nil {
				return nil, err
			}
		}
	}
	return p, nil
}

func build(name string) {
	err := runCommand("go", "build", "-o", name, toPath(".", "src"))
	if err != nil {
		os.Exit(1)
	}
}

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	err = os.Setenv("GOPATH", wd)
	if err != nil {
		log.Fatal(err)
	}

	switch getArg(1) {
	case "new":
		isLib := flag.Bool("lib", false, "create a library")
		flag.CommandLine.Parse(os.Args[2:])
		name := flag.Arg(0)
		newPackage(name, *isLib)
	case "test":
		getDeps(".")
		flag.CommandLine.Parse(os.Args[2:])
		path := flag.Arg(0)
		err := runCommand("go", "test", toPath(".", "src", path))
		if err != nil {
			os.Exit(1)
		}
	case "run":
		p, err := getDeps(".")
		if err != nil {
			log.Fatal(err)
		}
		build(p.Name)
		// run the compiled program with given arguments
		err = runCommand(append([]string{toPath(".", p.Name)}, os.Args[2:]...)...)
		if err != nil {
			log.Fatal(err)
		}
	case "build":
		p, err := getDeps(".")
		if err != nil {
			log.Fatal(err)
		}
		build(p.Name)
	default:
		printHelp()
	}
}
