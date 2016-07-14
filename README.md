# Toyen

Toyen is a package build system for Fuchsia. Similarly to other package build
systems, Toyen itself does not build the packages themselves. Toyen is
responsible for invoking the build system of individual projects, handling
dependencies across projects and packaging build artifacts.

## Usage

After checking out the source, you need to run `toyen` executable to generate
the `build.ninja` file.

```bash
$ toyen -src . -out out packages/root.bp
```

This is only needed after checking out the source for the first time. The
Ninja files have rules re-generating themselves when necessary.

Therefore, a typical iteration loop is:

1. edit your source
2. run `ninja -C out`

## Ninja and Blueprint

Toyen build system is implemented on top of Blueprint and Ninja, so the reader
should first understand how they work. This document provides a brief surface
exploration of the Blueprint and Ninja.

### Ninja

Ninja provides a simple syntax for defining:
1. file dependencies (e.g. foo.o depends on foo.c), and
2. arbitrary commands to run for generating files when they're out of date.

To keep things fast, there are very few other features. There's no branching
logic, globbing, or decision making of any kind, really.  Ninja files can be
written and read by humans, but that would be pointlessly tedious. Instead,
a program is used to generate Ninja files.

Check out the Ninja [documentation](http://martine.github.io/ninja/manual.html)
and [source](https://github.com/martine/ninja) for more information.

### Blueprint

Unlike other build systems (even those that also generate Ninja files, like
[gn](https://chromium.googlesource.com/chromium/src/tools/gn/)) Blueprint is not
a stand-alone executable.  You don't *run Blueprint*.  Blueprint also doesn't
come with any pre-defined build rules, except for the two that are necessary to
build Blueprint itself.

Blueprint is a set of Go packages that you use to implement whatever build rules
your project finds useful. Through these packages, Blueprint provides high
level constructs that are useful for generating Ninja files. Blueprint also
provides a parser for reading Blueprint config files.

To use Blueprint, you implement two things: your build modules, and a builder.
Blueprint comes with basic versions of both of these things that are capable of
building Blueprint itself. To learn more, read the extensive comments in the
[Blueprint code](http://github.com/google/blueprint) for complete documentation.

## FAQ

### Why the name "Toyen"?

Toyen is named after Marie Čermínová, known as
[Toyen](https://en.wikipedia.org/wiki/Toyen), Czech surrealist painter and
illustrator.
