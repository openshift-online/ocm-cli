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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/output"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "list",
	Short: "List ocm plugins",
	Long:  "List all the plugins under the user executable path",
	Args:  cobra.NoArgs,
	RunE:  run,
}

var args struct {
	columns  string
	nameOnly bool
}

func init() {
	fs := Cmd.Flags()
	fs.BoolVar(
		&args.nameOnly,
		"nameonly",
		false,
		"Show the plugin name only. This option is deprecated, use '--columns name' instead.",
	)
	fs.StringVar(
		&args.columns,
		"columns",
		"name, path",
		"Comma separated list of columns to display.",
	)
}

func run(cmd *cobra.Command, argv []string) error {
	// Create a context:
	ctx := context.Background()

	// If the deprecated --nameonly option has been used translate it into the corresponding
	// --columns option:
	if args.nameOnly {
		args.columns = "name"
	}

	// Load the configuration:
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Find the plugins:
	plugins, err := findPlugins()
	if err != nil {
		return err
	}

	// Create the output printer:
	printer, err := output.NewPrinter().
		Writer(os.Stdout).
		Pager(cfg.Pager).
		Build(ctx)
	if err != nil {
		return err
	}
	defer printer.Close()

	// Create the output table:
	table, err := printer.NewTable().
		Name("plugins").
		Columns(args.columns).
		Build(ctx)
	if err != nil {
		return err
	}
	defer table.Close()

	// Write the column headers:
	err = table.WriteHeaders()
	if err != nil {
		return err
	}

	// Write the rows:
	for _, plugin := range plugins {
		err = table.WriteRow(plugin)
		if err != nil {
			break
		}
	}
	if err != nil {
		return err
	}

	return nil
}

// Plugin contains the description fo a Plugin.
type Plugin struct {
	Name string
	Path string
}

// findPlugins scans the directories listed in the `PATH` environment variable looking for
// files that are plugins.
func findPlugins() (result []Plugin, err error) {
	defaultPath := filepath.SplitList(os.Getenv("PATH"))
	newPath := uniquePath(defaultPath)

	for _, dir := range newPath {
		_, err = os.Stat(dir)
		if os.IsNotExist(err) {
			err = nil
			continue
		}
		if err != nil {
			return
		}
		var list []Plugin
		list, err = listPlugins(dir)
		if err != nil {
			return
		}
		result = append(result, list...)
	}
	return
}

// listPlugins scans the given directory looking for files that are plugins.
func listPlugins(dir string) (result []Plugin, err error) {
	items, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		name := item.Name()
		if !strings.HasPrefix(name, pluginPrefix) {
			continue
		}
		path := filepath.Join(dir, name)
		var exec bool
		exec, err = isExecutable(path)
		if err != nil {
			return
		}
		if !exec {
			fmt.Printf("Warning: %s identified as an ocm plugin, but it is not executable.\n", path)
		}
		plugin := Plugin{
			Name: name,
			Path: dir,
		}
		result = append(result, plugin)
	}
	return
}

// uniquePath remove the duplicate items from the PATH
func uniquePath(path []string) []string {
	keys := make(map[string]int)
	uniPath := make([]string, 0)

	for _, p := range path {
		if p == "" {
			p = "."
		}
		keys[p] = 1
	}

	for element := range keys {
		uniPath = append(uniPath, element)
	}

	sort.Strings(uniPath)

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

// pluginPrefix is the prefix that plugin file names should have.
const pluginPrefix = "ocm-"
