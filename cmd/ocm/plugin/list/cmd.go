/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

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

package plugin

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/table"
	"github.com/spf13/cobra"
)

// Cmd represents the plugin command
var Cmd = &cobra.Command{
	Use:   "list",
	Short: "list ocm plugins",
	Long:  "list all the plugins under the user executable path",
	RunE:  run,
}

var args struct {
	nameOnly bool
}

func init() {
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.nameOnly,
		"nameonly",
		false,
		"Show the plugin name only",
	)
}

func run(cmd *cobra.Command, argv []string) error {
	defaultPath := filepath.SplitList(os.Getenv("PATH"))
	newPath := uniquePath(defaultPath)
	pluginPrefix := "ocm-"

	columns := "NAME,PATH"
	paddingByColumn := []int{20, 30}

	table.PrintPadded(os.Stdout, strings.Split(columns, ","), paddingByColumn)

	for _, dir := range newPath {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		items, err := ioutil.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, f := range items {
			if f.IsDir() {
				continue
			}

			if !strings.HasPrefix(f.Name(), pluginPrefix) {
				continue
			}

			plugin := f.Name()

			var pluginOutput []string
			absPath := dir + "/" + plugin
			if isExec, err := isExecutable(absPath); err == nil && !isExec {
				defer fmt.Printf("Warning: %s identified as an ocm plugin, but it is not executable. \n", absPath)
			} else if err != nil {
				return err
			} else {
				if args.nameOnly {
					pluginOutput = []string{plugin}
				} else {
					pluginOutput = []string{plugin, dir}
				}
				table.PrintPadded(os.Stdout, pluginOutput, paddingByColumn)
			}
		}
	}
	return nil
}

// uniquePath remove the duplicate items from the PATH
func uniquePath(path []string) []string {
	keys := make(map[string]int)
	uniPath := make([]string, 0)

	for _, p := range path {
		keys[p] = 1
	}

	for element := range keys {
		uniPath = append(uniPath, element)
	}

	return uniPath
}

// detect if the plugin is excutable
func isExecutable(file string) (bool, error) {
	info, err := os.Stat(file)
	if err != nil {
		return false, err
	}

	if runtime.GOOS == "windows" {
		fileExt := strings.ToLower(filepath.Ext(file))

		switch fileExt {
		case ".bat", ".cmd", ".com", ".exe", ".ps1":
			return true, nil
		}
		return false, nil
	}

	if m := info.Mode(); !m.IsDir() && m&0111 != 0 {
		return true, nil
	}

	return false, nil
}
