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
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"packages/yaml"
	"strings"
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

func blueText(s string) string {
	return "\033[34;1m" + s + "\033[0m"
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

func getDeps(cfg string) (*gopkgCfg, error) {
	buf, err := ioutil.ReadFile(cfg)
	if err != nil {
		return nil, err
	}
	var p = new(gopkgCfg)
	err = yaml.Unmarshal(buf, p)
	if err != nil {
		return nil, err
	}

	tempDir := toPath(os.TempDir(), "gopkg"+randomStr())
	for _, pkg := range p.Packages {
		fmt.Println(greenText("Getting"), pkg.Name, "["+pkg.Git+"]")
		if pkg.Branch != "" {
			fmt.Println("  - Branch:", pkg.Branch)
		}
		if pkg.Tag != "" {
			fmt.Println("  - Tag:", pkg.Tag)
		}
		if pkg.Rev != "" {
			fmt.Println("  - Rev:", pkg.Rev)
		}

		pkgDir := toPath("src", "packages", pkg.Name)
		if dirExists(pkgDir) {
			fmt.Println("  - " + blueText("Already exists") + "\n")
			continue
		} else {
			gitPath := toPath(tempDir, pkg.Name)
			err := runCommand("git", "clone", "-q", pkg.Git, gitPath)
			if err != nil {
				os.Exit(1)
			}
			if pkg.Branch != "" {
				err = runCommandInDir(gitPath, "git", "checkout", "-q", pkg.Branch)
				if err != nil {
					os.Exit(1)
				}
			}
			if pkg.Tag != "" {
				err = runCommandInDir(gitPath, "git", "checkout", "-q", pkg.Tag)
				if err != nil {
					os.Exit(1)
				}
			}
			if pkg.Rev != "" {
				err = runCommandInDir(gitPath, "git", "reset", "-q", "--hard", pkg.Rev)
				if err != nil {
					os.Exit(1)
				}
			}

			pkgPath := toPath("src", "packages", pkg.Name)
			if fileExists(toPath(gitPath, "gopkg.yaml")) {
				err = copyDir(toPath(gitPath, "src"), pkgPath)
				if err != nil {
					return nil, err
				}
				err = filepath.Walk(gitPath, func(path string, f os.FileInfo, err error) error {
					if f == nil {
						return err
					}
					if f.IsDir() {
						return nil
					} else {
						_, err := copyFile(path, toPath(pkgPath, f.Name()))
						return err
					}
				})
				if err != nil {
					return nil, err
				}
				fmt.Println("  - " + greenText("Done") + "\n")

				_, err := getDeps(toPath(gitPath, "gopkg.yaml"))
				if err != nil {
					return nil, err
				}
			} else {
				err = copyDir(gitPath, pkgPath)
				if err != nil {
					return nil, err
				}
				os.RemoveAll(toPath(pkgPath, ".git"))
				fmt.Println("  - "+greenText("Done"), "["+yellowText("Not used GoPKG")+"]\n")

				// TODO: 分析非 gopkg 包的依赖并安装
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
		getDeps("gopkg.yaml")
		flag.CommandLine.Parse(os.Args[2:])
		path := flag.Arg(0)
		err = runCommand("go", "test", toPath(".", "src", path))
		if err != nil {
			os.Exit(1)
		}
	case "run":
		p, err := getDeps("gopkg.yaml")
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
		p, err := getDeps("gopkg.yaml")
		if err != nil {
			log.Fatal(err)
		}
		build(p.Name)
	default:
		printHelp()
	}
}
