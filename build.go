// Copyright 2016 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"
)

var (
	pctx = blueprint.NewPackageContext("fuchsia.googlesource.com/toyen")

	cmakeCmd = pctx.StaticVariable("cmakeCmd", "cmake")
	gnCmd    = pctx.StaticVariable("gnCmd", "gn")
	makeCmd  = pctx.StaticVariable("makeCmd", "make")
	ninjaCmd = pctx.StaticVariable("ninjaCmd", "ninja")

	cmake = pctx.StaticRule("cmake",
		blueprint.RuleParams{
			Command: "cd $buildDir && $envVars $cmakeCmd -GNinja " +
				"$cmakeOptions $cmakeDir",
			Generator:   true,
			Description: "cmake $cmakeDir",
		},
		"envVars", "cmakeOptions", "cmakeDir", "buildDir")

	gn = pctx.StaticRule("gn",
		blueprint.RuleParams{
			Command: "$envVars $gnCmd gen $buildDir " +
				"--root=$gnDir --script-executable=/usr/bin/env --args='$gnArgs'",
			Generator:   true,
			Description: "gn $gnDir",
		},
		"envVars", "gnDir", "gnArgs", "buildDir")

	_make = pctx.StaticRule("make",
		blueprint.RuleParams{
			Command: "$envVars $makeCmd -j $Jobs " +
				"-C $makeDir -f $makeFile $targets",
			Description: "make $makeDir",
		},
		"envVars", "targets", "makeFile", "makeDir")

	ninja = pctx.StaticRule("ninja",
		blueprint.RuleParams{
			Command: "$envVars $ninjaCmd -j $Jobs " +
				"-C $ninjaDir -f $ninjaFile $targets",
			Description: "ninja $ninjaDir",
		},
		"envVars", "targets", "ninjaFile", "ninjaDir")

	script = pctx.StaticRule("script",
		blueprint.RuleParams{
			Command:     "cd $workingDir && $envVars $scriptCmd $scriptArgs",
			Description: "sh $in",
		},
		"envVars", "scriptCmd", "scriptArgs", "workingDir")

	cp = pctx.StaticRule("cp",
		blueprint.RuleParams{
			Command:     "cp -vR $in $out",
			Description: "cp $out",
		})

	install = pctx.StaticRule("install",
		blueprint.RuleParams{
			Command:     "install -c $in $out",
			Description: "install $out",
		})

	mkdir = pctx.StaticRule("mkdir",
		blueprint.RuleParams{
			Command:     "mkdir -p $out",
			Description: "mkdir $out",
		})

	rm = pctx.StaticRule("rm",
		blueprint.RuleParams{
			Command:     "rm -rf $files",
			Description: "rm $out",
		},
		"files")

	stamp = pctx.StaticRule("stamp",
		blueprint.RuleParams{
			Command:     "touch $out",
			Description: "stamp $out",
		})
)

type BuilderModule interface {
	TargetName() string
}

type builderModule struct {
	targetName string
}

func (bm *builderModule) TargetName() string {
	return bm.targetName
}

type Alias struct {
	builderModule
	properties struct{}
	config     Config
}

func newAliasModuleFactory(config Config) func() (blueprint.Module, []interface{}) {
	return func() (blueprint.Module, []interface{}) {
		module := &Alias{
			config: config,
		}
		return module, []interface{}{&module.properties}
	}
}

func (a *Alias) GenerateBuildActions(ctx blueprint.ModuleContext) {
	a.targetName = ctx.ModuleName()

	ctx.Build(pctx, blueprint.BuildParams{
		Rule:    blueprint.Phony,
		Outputs: []string{ctx.ModuleName()},
		Inputs:  getDirectDependencies(ctx),
	})
}

type Clean struct {
	builderModule
	properties struct {
		Dirs []string
	}
	config Config
}

func newCleanModuleFactory(config Config) func() (blueprint.Module, []interface{}) {
	return func() (blueprint.Module, []interface{}) {
		module := &Clean{
			config: config,
		}
		return module, []interface{}{&module.properties}
	}
}

func (c *Clean) GenerateBuildActions(ctx blueprint.ModuleContext) {
	c.targetName = ctx.ModuleName()

	if len(c.properties.Dirs) != 0 {
		// Add a rule for deleting all the specified files
		ctx.Build(pctx, blueprint.BuildParams{
			Rule:    rm,
			Outputs: []string{ctx.ModuleName()},
			Args: map[string]string{
				"files": strings.Join(c.properties.Dirs, " "),
			},
			Implicits: getDirectDependencies(ctx),
		})
	} else {
		ctx.Build(pctx, blueprint.BuildParams{
			Rule:      blueprint.Phony,
			Outputs:   []string{ctx.ModuleName()},
			Implicits: getDirectDependencies(ctx),
		})
	}
}

type CMake struct {
	builderModule
	properties struct {
		Env      []string
		Options  []string
		Src      string
		BuildDir string
	}
	config Config
}

func newCMakeModuleFactory(config Config) func() (blueprint.Module, []interface{}) {
	return func() (blueprint.Module, []interface{}) {
		module := &CMake{
			config: config,
		}
		return module, []interface{}{&module.properties}
	}
}

func (c *CMake) GenerateBuildActions(ctx blueprint.ModuleContext) {
	c.targetName = ctx.ModuleName()

	options := make([]string, len(c.properties.Options))
	for i := range c.properties.Options {
		options[i] = fmt.Sprintf("-D%s", c.properties.Options[i])
	}

	// Add a rule for making the destination directory, in case it doesn't exist.
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:     mkdir,
		Outputs:  []string{c.properties.BuildDir},
		Optional: true,
	})

	cmakeArgs := map[string]string{
		"cmakeOptions": strings.Join(options, " "),
		"cmakeDir":     c.properties.Src,
		"envVars":      strings.Join(c.properties.Env, " "),
		"buildDir":     c.properties.BuildDir,
	}

	ninjaFile := filepath.Join(c.properties.BuildDir, "build.ninja")

	// Add a rule to generate the Ninja file.
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      cmake,
		Outputs:   []string{ninjaFile},
		Implicits: getDirectDependencies(ctx),
		OrderOnly: []string{c.properties.BuildDir},
		Args:      cmakeArgs,
	})
}

type Copy struct {
	builderModule
	properties struct {
		Sources     []string
		Destination string
	}
	config Config
}

func newCopyModuleFactory(config Config) func() (blueprint.Module, []interface{}) {
	return func() (blueprint.Module, []interface{}) {
		module := &Copy{
			config: config,
		}
		return module, []interface{}{&module.properties}
	}
}

func (c *Copy) GenerateBuildActions(ctx blueprint.ModuleContext) {
	c.targetName = ctx.ModuleName()

	// If multiple source files, destination is a directory, otherwise, it's a file.
	var destinationDir string
	destinationFiles := make([]string, len(c.properties.Sources))
	if len(c.properties.Sources) > 1 {
		destinationDir = c.properties.Destination
		for i, src := range c.properties.Sources {
			destinationFiles[i] = filepath.Join(destinationDir, filepath.Base(src))
		}
	} else {
		destinationDir = filepath.Dir(c.properties.Destination)
		destinationFiles[0] = c.properties.Destination
	}

	// Add a rule for making the destination directory, in case it doesn't exist.
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:     mkdir,
		Outputs:  []string{destinationDir},
		Optional: true,
	})

	// Add a rule for each source/destination pair.
	for i, src := range c.properties.Sources {
		ctx.Build(pctx, blueprint.BuildParams{
			Rule:      cp,
			Outputs:   []string{destinationFiles[i]},
			Inputs:    []string{src},
			Implicits: getDirectDependencies(ctx),
			OrderOnly: []string{destinationDir},
			Optional:  true,
		})
	}

	// Add a phony rule to copy all the files with one rule.
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:    blueprint.Phony,
		Outputs: []string{ctx.ModuleName()},
		Inputs:  destinationFiles,
	})
}

type Gn struct {
	builderModule
	properties struct {
		Env      []string
		Args     []string
		SrcDir   string
		BuildDir string
	}
	config Config
}

func newGnModuleFactory(config Config) func() (blueprint.Module, []interface{}) {
	return func() (blueprint.Module, []interface{}) {
		module := &Gn{
			config: config,
		}
		return module, []interface{}{&module.properties}
	}
}

func (g *Gn) GenerateBuildActions(ctx blueprint.ModuleContext) {
	g.targetName = ctx.ModuleName()

	gnArgs := map[string]string{
		"envVars":  strings.Join(g.properties.Env, " "),
		"gnDir":    g.properties.SrcDir,
		"gnArgs":   strings.Join(g.properties.Args, " "),
		"buildDir": g.properties.BuildDir,
	}

	ninjaFile := filepath.Join(g.properties.BuildDir, "build.ninja")

	// Add a rule to generate the Ninja file.
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      gn,
		Outputs:   []string{ninjaFile},
		Implicits: getDirectDependencies(ctx),
		Args:      gnArgs,
	})
}

type Install struct {
	builderModule
	properties struct {
		Sources     []string
		Destination string
	}
	config Config
}

func newInstallModuleFactory(config Config) func() (blueprint.Module, []interface{}) {
	return func() (blueprint.Module, []interface{}) {
		module := &Install{
			config: config,
		}
		return module, []interface{}{&module.properties}
	}
}

func (i *Install) GenerateBuildActions(ctx blueprint.ModuleContext) {
	i.targetName = ctx.ModuleName()

	destinationFiles := make([]string, len(i.properties.Sources))
	for j, src := range i.properties.Sources {
		destinationFiles[j] = filepath.Join(i.properties.Destination, filepath.Base(src))
	}

	// Add a rule for making the destination directory, in case it doesn't exist.
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:     mkdir,
		Outputs:  []string{i.properties.Destination},
		Optional: true,
	})

	// Add a rule to install sources.
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      install,
		Outputs:   destinationFiles,
		Inputs:    i.properties.Sources,
		Implicits: getDirectDependencies(ctx),
		OrderOnly: []string{i.properties.Destination},
	})
}

type Make struct {
	builderModule
	properties struct {
		Env      []string
		Makefile string
		Targets  []string
		Outputs  []string
	}
	config Config
}

func newMakeModuleFactory(config Config) func() (blueprint.Module, []interface{}) {
	return func() (blueprint.Module, []interface{}) {
		module := &Make{
			config: config,
		}
		return module, []interface{}{&module.properties}
	}
}

func (m *Make) GenerateBuildActions(ctx blueprint.ModuleContext) {
	m.targetName = ctx.ModuleName()

	makeArgs := map[string]string{
		"envVars":  strings.Join(m.properties.Env, " "),
		"makeFile": filepath.Base(m.properties.Makefile),
		"makeDir":  filepath.Dir(m.properties.Makefile),
	}
	if len(m.properties.Targets) > 0 {
		makeArgs["targets"] = strings.Join(m.properties.Targets, " ")
	}

	var outputs []string
	if len(m.properties.Outputs) > 0 {
		outputs = m.properties.Outputs
	} else {
		outputs = []string{ctx.ModuleName()}
	}

	// Add a single rule where our touchFile depends on our make rule.
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      _make,
		Outputs:   outputs,
		Implicits: append(getDirectDependencies(ctx), m.properties.Makefile),
		Args:      makeArgs,
		Optional:  true,
	})
}

type Ninja struct {
	builderModule
	properties struct {
		Env       []string
		NinjaFile string
		Targets   []string
		Outputs   []string
	}
	config Config
}

func newNinjaModuleFactory(config Config) func() (blueprint.Module, []interface{}) {
	return func() (blueprint.Module, []interface{}) {
		module := &Ninja{
			config: config,
		}
		return module, []interface{}{&module.properties}
	}
}

func (n *Ninja) GenerateBuildActions(ctx blueprint.ModuleContext) {
	n.targetName = ctx.ModuleName()

	ninjaArgs := map[string]string{
		"envVars":   strings.Join(n.properties.Env, " "),
		"ninjaFile": filepath.Base(n.properties.NinjaFile),
		"ninjaDir":  filepath.Dir(n.properties.NinjaFile),
	}
	if len(n.properties.Targets) != 0 {
		ninjaArgs["targets"] = strings.Join(n.properties.Targets, " ")
	}

	var outputs []string
	if len(n.properties.Outputs) > 0 {
		outputs = n.properties.Outputs
	} else {
		outputs = []string{ctx.ModuleName()}
	}

	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      ninja,
		Outputs:   outputs,
		Implicits: append(getDirectDependencies(ctx), n.properties.NinjaFile),
		Args:      ninjaArgs,
		Optional:  true,
	})
}

type Script struct {
	builderModule
	properties struct {
		Script     string
		Outputs    []string
		Inputs     []string
		Args       []string
		WorkingDir string
		Env        []string
		GenFiles   []string
	}
	config Config
}

func newScriptModuleFactory(config Config) func() (blueprint.Module, []interface{}) {
	return func() (blueprint.Module, []interface{}) {
		module := &Script{
			config: config,
		}
		return module, []interface{}{&module.properties}
	}
}

func (s *Script) GenerateBuildActions(ctx blueprint.ModuleContext) {
	s.targetName = ctx.ModuleName()

	// Add a rule for making the destination directory, in case it doesn't exist.
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:     mkdir,
		Outputs:  []string{s.properties.WorkingDir},
		Optional: true,
	})

	ruleArgs := map[string]string{
		"envVars":    strings.Join(s.properties.Env, " "),
		"scriptCmd":  s.properties.Script,
		"scriptArgs": strings.Join(s.properties.Args, " "),
		"workingDir": s.properties.WorkingDir,
	}

	implicits := append(
		s.properties.Inputs,
		append(getDirectDependencies(ctx), s.properties.Script)...,
	)

	outputs := append(s.properties.Outputs, s.properties.GenFiles...)

	if len(outputs) != 0 {
		// Add a rule to run the script to generate the outputs.
		ctx.Build(pctx, blueprint.BuildParams{
			Rule:      script,
			Outputs:   outputs,
			Inputs:    []string{s.properties.Script},
			Args:      ruleArgs,
			Implicits: implicits,
			OrderOnly: []string{s.properties.WorkingDir},
			Optional:  true,
		})

		// The only default rule should be one with the module name.
		ctx.Build(pctx, blueprint.BuildParams{
			Rule:    blueprint.Phony,
			Outputs: []string{ctx.ModuleName()},
			Inputs:  outputs,
		})
	} else {
		// For scripts with no outputs, use a script rule with the module name.
		ctx.Build(pctx, blueprint.BuildParams{
			Rule:      script,
			Outputs:   []string{ctx.ModuleName()},
			Inputs:    []string{s.properties.Script},
			Args:      ruleArgs,
			Implicits: implicits,
			OrderOnly: []string{s.properties.WorkingDir},
		})
	}
}

type Bootstrap struct {
	config Config
}

func newBootstrapFactory(config Config) func() blueprint.Singleton {
	return func() blueprint.Singleton {
		return &Bootstrap{
			config: config,
		}
	}
}

func (m *Bootstrap) GenerateBuildActions(ctx blueprint.SingletonContext) {
	executable, _ := exec.LookPath(os.Args[0])

	builder := ctx.Rule(pctx, "builder",
		blueprint.RuleParams{
			Command:     fmt.Sprintf("%s $rootBlueprintFile", executable),
			Description: "Regenerating Ninja files",
			Generator:   true,
			Depfile:     depFile,
		}, "rootBlueprintFile")

	args := map[string]string{
		"rootBlueprintFile": rootBlueprintsFile,
	}

	ctx.Build(pctx, blueprint.BuildParams{
		Rule:    builder,
		Outputs: []string{outFile},
		Args:    args,
	})
}

func isBuilderModule(m blueprint.Module) bool {
	_, ok := m.(BuilderModule)
	return ok
}

func getDirectDependencies(ctx blueprint.ModuleContext) []string {
	var depTargets []string
	ctx.VisitDirectDepsIf(isBuilderModule, func(m blueprint.Module) {
		target := m.(BuilderModule)
		depTargets = append(depTargets, target.TargetName())
	})
	return depTargets
}

func getAllDependencies(ctx blueprint.ModuleContext) []string {
	var depTargets []string
	ctx.VisitDepsDepthFirstIf(isBuilderModule, func(m blueprint.Module) {
		target := m.(BuilderModule)
		depTargets = append(depTargets, target.TargetName())
	})
	return depTargets
}
