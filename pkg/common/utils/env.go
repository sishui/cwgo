/*
 * Copyright 2022 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"bytes"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/cwgo/pkg/consts"
	"github.com/cloudwego/kitex/tool/internal_pkg/log"
)

func GetGOPATH() (gopath string, err error) {
	ps := filepath.SplitList(os.Getenv(consts.GOPATH))
	if len(ps) > 0 {
		gopath = ps[0]
	}
	if gopath == "" {
		cmd := exec.Command(consts.Go, consts.Env, consts.GOPATH)
		var out bytes.Buffer
		cmd.Stderr = &out
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			gopath = strings.Trim(out.String(), " \t\n\r")
		}
	}
	if gopath == "" {
		ps := GetBuildGoPaths()
		if len(ps) > 0 {
			gopath = ps[0]
		}
	}
	isExist, err := PathExist(gopath)
	if !isExist {
		return "", err
	}
	return strings.Replace(gopath, consts.Slash, string(os.PathSeparator), -1), nil
}

// GetBuildGoPaths returns the list of Go path directories.
func GetBuildGoPaths() []string {
	var all []string
	for _, p := range filepath.SplitList(build.Default.GOPATH) {
		if p == "" || p == build.Default.GOROOT {
			continue
		}
		if strings.HasPrefix(p, consts.Tilde) {
			continue
		}
		all = append(all, p)
	}
	for k, v := range all {
		if strings.HasSuffix(v, consts.Slash) || strings.HasSuffix(v, string(os.PathSeparator)) {
			v = v[:len(v)-1]
		}
		all[k] = v
	}
	return all
}

var goModReg = regexp.MustCompile(`^\s*module\s+(\S+)\s*`)

// SearchGoMod searches go.mod from the given directory (which must be an absolute path) to
// the root directory. When the go.mod is found, its module name and path will be returned.
func SearchGoMod(cwd string, recurse bool) (moduleName, path string, found bool) {
	for {
		path = filepath.Join(cwd, consts.GoMod)
		data, err := ioutil.ReadFile(path)
		if err == nil {
			for _, line := range strings.Split(string(data), consts.LineBreak) {
				m := goModReg.FindStringSubmatch(line)
				if m != nil {
					return m[1], cwd, true
				}
			}
			return fmt.Sprintf("<module name not found in '%s'>", path), path, true
		}

		if !os.IsNotExist(err) {
			return
		}
		if !recurse {
			break
		}
		cwd = filepath.Dir(cwd)
		// the root directory will return itself by using "filepath.Dir()"; to prevent dead loops, so jump out
		if cwd == filepath.Dir(cwd) {
			break
		}
	}
	return
}

func InitGoMod(module string) error {
	isExist, err := PathExist(consts.GoMod)
	if err != nil {
		return err
	}
	if isExist {
		return nil
	}
	gg, err := exec.LookPath(consts.Go)
	if err != nil {
		return err
	}
	cmd := &exec.Cmd{
		Path:   gg,
		Args:   []string{consts.Go, consts.Mod, consts.Init, module},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return cmd.Run()
}

// IsWindows determines whether the current operating system is Windows
func IsWindows() bool {
	return consts.SysType == consts.WindowsOS
}

func ReplaceThriftVersion(idlType string) {
	if idlType == consts.Thrift {
		cmd := "go mod edit -replace github.com/apache/thrift=github.com/apache/thrift@v0.13.0"
		argv := strings.Split(cmd, consts.BlackSpace)
		err := exec.Command(argv[0], argv[1:]...).Run()

		res := "Done"
		if err != nil {
			res = err.Error()
		}
		log.Warn("Adding apache/thrift@v0.13.0 to go.mod for generated code ..........", res)
	}
}
