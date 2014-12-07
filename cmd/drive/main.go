// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package contains the main entry point of gd.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rakyll/command"
	"github.com/rakyll/drive"
	"github.com/rakyll/drive/config"
)

var context *config.Context

const (
	descInit    = "inits a directory and authenticates user"
	descPull    = "pulls remote changes from google drive"
	descPush    = "push local changes to google drive"
	descDiff    = "compares a local file with remote"
	descPublish = "publishes a file and prints its publicly available url"
)

func main() {
	command.On("init", descInit, &initCmd{}, []string{})
	command.On("pull", descPull, &pullCmd{}, []string{})
	command.On("push", descPush, &pushCmd{}, []string{})
	command.On("diff", descDiff, &diffCmd{}, []string{})
	command.On("pub", descPublish, &publishCmd{}, []string{})
	command.ParseAndRun()
}

type initCmd struct{}

func (cmd *initCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (cmd *initCmd) Run(args []string) {
	exitWithError(drive.New(initContext(args), nil).Init())
}

type pullCmd struct {
	isRecursive *bool
	isNoPrompt  *bool
}

func (cmd *pullCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.isRecursive = fs.Bool("r", true, "performs the pull action recursively")
	cmd.isNoPrompt = fs.Bool("no-prompt", false, "shows no prompt before applying the pull action")
	return fs
}

func (cmd *pullCmd) Run(args []string) {
	context, path := discoverContext(args)
	exitWithError(drive.New(context, &drive.Options{
		Path:        path,
		IsRecursive: *cmd.isRecursive,
		IsNoPrompt:  *cmd.isNoPrompt,
	}).Pull())
}

type pushCmd struct {
	hidden      *bool
	isNoPrompt  *bool
	isRecursive *bool
	mountedPush *bool
}

func (cmd *pushCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool("hidden", false, "allows syncing of hidden paths")
	cmd.isRecursive = fs.Bool("r", true, "performs the push action recursively")
	cmd.isNoPrompt = fs.Bool("no-prompt", false, "shows no prompt before applying the push action")
	cmd.mountedPush = fs.Bool("m", false, "allows pushing of mounted paths")
	return fs
}

func (cmd *pushCmd) Run(args []string) {
	argc := len(args)
	var contextArgs, rest, sources []string
	if *cmd.mountedPush && argc >= 1 {
		rest = args[:argc-1]
		contextArgs = args[argc-1:]
	} else {
		contextArgs = args
		sources = make([]string, argc)
		for i, path := range args {
			sources[i] = strings.Join([]string{"/", path}, "")
		}
	}

	context, path := discoverContext(contextArgs)
	contextAbsPath, err := filepath.Abs(path)
	exitWithError(err)

	mountPoints, auxSrcs := config.MountPoints(path, contextAbsPath, rest, *cmd.hidden)
	sources = append(sources, auxSrcs...)

	exitWithError(drive.New(context, &drive.Options{
		Path:        path,
		Hidden:      *cmd.hidden,
		IsNoPrompt:  *cmd.isNoPrompt,
		IsRecursive: *cmd.isRecursive,
		Mounts:      mountPoints,
		Sources:     sources,
	}).Push())
}

type diffCmd struct{}

func (cmd *diffCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (cmd *diffCmd) Run(args []string) {
	context, path := discoverContext(args)
	exitWithError(drive.New(context, &drive.Options{
		Path: path,
	}).Diff())
}

type publishCmd struct{}

func (cmd *publishCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (cmd *publishCmd) Run(args []string) {
	context, path := discoverContext(args)
	exitWithError(drive.New(context, &drive.Options{
		Path: path,
	}).Publish())
}

func initContext(args []string) *config.Context {
	var err error
	context, err = config.Initialize(getContextPath(args))
	exitWithError(err)
	return context
}

func discoverContext(args []string) (*config.Context, string) {
	var err error
	context, err = config.Discover(getContextPath(args))
	exitWithError(err)
	relPath := ""
	if len(args) > 0 {
		relPath, err = filepath.Rel(context.AbsPath, args[0])
	}
	exitWithError(err)
	return context, relPath
}

func getContextPath(args []string) (contextPath string) {
	if len(args) > 0 {
		contextPath = args[0]
	}
	if contextPath == "" {
		contextPath, _ = os.Getwd()
	}
	return
}

func exitWithError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
