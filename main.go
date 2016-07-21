// Copyright 2016 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/blueprint"
	"github.com/google/blueprint/deptools"
)

var (
	rootBlueprintsFile string
	manifestFile       string

	depFile      string
	outFile      string
	hostTriple   string
	targetTriple string

	srcDir string
	outDir string
)

func init() {
	flag.StringVar(&srcDir, "src", ".", "the source directory")
	flag.StringVar(&outDir, "out", ".", "the build output directory")
	flag.StringVar(&hostTriple, "host", triple(), "build tools to run on")
	flag.StringVar(&targetTriple, "target", "", "target triple")

	flag.Usage = func() {
	    fmt.Printf("Usage: toyen [options] <Blueprint file>\n")
	    flag.PrintDefaults()
	}
}

func triple() string {
	arches := map[string]string{
		"386":   "i386",
		"amd64": "x86_64",
		"arm":   "armv7a",
		"arm64": "aarch64",
	}

	var arch string
	var ok bool
	if arch, ok = arches[runtime.GOARCH]; !ok {
		arch = "unknown"
	}

	return fmt.Sprintf("%s-%s", arch, runtime.GOOS)
}

func main() {
	flag.Parse()

	// The top-level Blueprints file is passed as the first argument.
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	rootBlueprintsFile, _ = filepath.Abs(flag.Arg(0))

	srcDir, _ = filepath.Abs(srcDir)
	outDir, _ = filepath.Abs(outDir)

	outFile = filepath.Join(outDir, "build.ninja")
	depFile = outFile + ".d"

	config := NewConfig(srcDir, outDir, hostTriple, targetTriple)

	// Create the build context.
	ctx := blueprint.NewContext()

	// Register custom module types.
	ctx.RegisterModuleType("alias", newAliasModuleFactory(config))
	ctx.RegisterModuleType("clean", newCleanModuleFactory(config))
	ctx.RegisterModuleType("cmake", newCMakeModuleFactory(config))
	ctx.RegisterModuleType("copy", newCopyModuleFactory(config))
	ctx.RegisterModuleType("gn", newGnModuleFactory(config))
	ctx.RegisterModuleType("install", newInstallModuleFactory(config))
	ctx.RegisterModuleType("make", newMakeModuleFactory(config))
	ctx.RegisterModuleType("ninja", newNinjaModuleFactory(config))
	ctx.RegisterModuleType("script", newScriptModuleFactory(config))

	ctx.RegisterSingletonType("bootstrap", newBootstrapFactory(config))

	deps, errs := ctx.ParseBlueprintsFiles(rootBlueprintsFile)
	if len(errs) > 0 {
		fatalErrors(errs)
	}

	errs = ctx.ResolveDependencies(config)
	if len(errs) > 0 {
		fatalErrors(errs)
	}

	extraDeps, errs := ctx.PrepareBuildActions(config)
	if len(errs) > 0 {
		fatalErrors(errs)
	}
	deps = append(deps, extraDeps...)

	buf := bytes.NewBuffer(nil)
	if err := ctx.WriteBuildFile(buf); err != nil {
		fatalf("error generating Ninja file contents: %s", err)
	}

	const outFilePermissions = 0666

	if err := ioutil.WriteFile(outFile, buf.Bytes(), outFilePermissions); err != nil {
		fatalf("error writing %s: %s", outFile, err)
	}

	if err := deptools.WriteDepFile(depFile, outFile, deps); err != nil {
		fatalf("error writing depfile: %s", err)
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Print("\n")
	os.Exit(1)
}

func fatalErrors(errs []error) {
	red := "\x1b[31m"
	unred := "\x1b[0m"

	for _, err := range errs {
		switch err := err.(type) {
		case *blueprint.Error:
			fmt.Printf("%serror:%s %s\n", red, unred, err.Error())
		default:
			fmt.Printf("%sinternal error:%s %s\n", red, unred, err)
		}
	}
	os.Exit(1)
}
