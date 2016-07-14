// Copyright 2016 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	OutDir       = pctx.VariableConfigMethod("OutDir", Config.OutDir)
	SrcDir       = pctx.VariableConfigMethod("SrcDir", Config.SrcDir)
	HostTriple   = pctx.VariableConfigMethod("HostTriple", Config.HostTriple)
	TargetTriple = pctx.VariableConfigMethod("TargetTriple", Config.TargetTriple)

	HostArch     = pctx.VariableConfigMethod("HostArch", Config.HostArch)
	HostOS       = pctx.VariableConfigMethod("HostOS", Config.HostOS)
	TargetArch   = pctx.VariableConfigMethod("TargetArch", Config.TargetArch)
	TargetOS     = pctx.VariableConfigMethod("TargetOS", Config.TargetOS)

	jobs int
	Jobs = pctx.VariableConfigMethod("Jobs", func(c *buildConfig) string {
		return fmt.Sprintf("%d", jobs)
	})

	ToolsDir = pctx.StaticVariable("ToolsDir", filepath.Join("root", "tools"))
)

func init() {
	flag.IntVar(&jobs, "j", runtime.NumCPU(), "number of parallel jobs")
}

type Config interface {
	OutDir() string
	SrcDir() string
	HostTriple() string
	TargetTriple() string
	HostArch() string
	HostOS() string
	TargetArch() string
	TargetOS() string
}

type buildConfig struct {
	srcDir       string
	outDir       string
	hostTriple   string
	targetTriple string
}

func NewConfig(srcDir string, outDir string, hostTriple string, targetTriple string) *buildConfig {
	return &buildConfig{srcDir, outDir, hostTriple, targetTriple}
}

func (c *buildConfig) OutDir() string {
	return c.outDir
}

func (c *buildConfig) SrcDir() string {
	return c.srcDir
}

func (c *buildConfig) HostTriple() string {
	return c.hostTriple
}

func (c *buildConfig) TargetTriple() string {
	return c.targetTriple
}

func (c *buildConfig) HostArch() string {
	s := strings.Split(c.hostTriple, "-")
	return s[0]
}

func (c *buildConfig) HostOS() string {
	s := strings.Split(c.hostTriple, "-")
	return strings.Title(s[len(s) - 1])
}

func (c *buildConfig) TargetArch() string {
	s := strings.Split(c.targetTriple, "-")
	return s[0]
}

func (c *buildConfig) TargetOS() string {
	s := strings.SplitN(c.targetTriple, "-", 3)
	return strings.Title(s[len(s) - 1])
}
