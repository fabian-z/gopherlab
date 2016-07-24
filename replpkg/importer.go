package replpkg

import (
	"go/types"
	"golang.org/x/tools/go/gcimporter15"
)

// Implementation in this file is taken from https://github.com/d4l3k/go-pry
// MIT License, Copyright (c) 2015 Tristan Rice
var gcImporter = gcimporter.Import

// importer implements go/types.Importer.
// It also implements go/types.ImporterFrom, which was new in Go 1.6,
// so vendoring will work.
type importer struct {
	impFn    func(packages map[string]*types.Package, path, srcDir string) (*types.Package, error)
	packages map[string]*types.Package
}

func (i importer) Import(path string) (*types.Package, error) {
	return i.impFn(i.packages, path, "")
}

// This is in its own file so it can be ignored under Go 1.5.

func (i importer) ImportFrom(path, srcDir string, mode types.ImportMode) (*types.Package, error) {
	return i.impFn(i.packages, path, srcDir)
}
