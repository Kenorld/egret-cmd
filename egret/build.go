package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/kenorld/eject-cmd/harness"
	"github.com/kenorld/eject-core"
)

var cmdBuild = &Command{
	UsageLine: "build [import path] [target path] [run mode]",
	Short:     "build a Eject application (e.g. for deployment)",
	Long: `
Build the Eject web application named by the given import path.
This allows it to be deployed and run on a machine that lacks a Go installation.

The run mode is used to select which set of app.yaml configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

WARNING: The target path will be completely deleted, if it already exists!

For example:

    eject build github.com/kenorld/eject-samples/chat /tmp/chat
`,
}

func init() {
	cmdBuild.Run = buildApp
}

func buildApp(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "%s\n%s", cmdBuild.UsageLine, cmdBuild.Long)
		return
	}

	appImportPath, destPath, mode := args[0], args[1], "dev"
	if len(args) >= 3 {
		mode = args[2]
	}

	if !eject.Initialized {
		eject.Init(mode, appImportPath, "")
	}

	// First, verify that it is either already empty or looks like a previous
	// build (to avoid clobbering anything)
	if exists(destPath) && !empty(destPath) && !exists(path.Join(destPath, "run.sh")) {
		errorf("Abort: %s exists and does not look like a build directory.", destPath)
	}

	os.RemoveAll(destPath)
	os.MkdirAll(destPath, 0777)

	app, eerr := harness.Build()
	panicOnError(eerr, "Failed to build")

	// Included are:
	// - run scripts
	// - binary
	// - eject
	// - app

	// Eject and the app are in a directory structure mirroring import path
	srcPath := path.Join(destPath, "src")
	destBinaryPath := path.Join(destPath, filepath.Base(app.BinaryPath))
	tmpEjectPath := path.Join(srcPath, filepath.FromSlash(eject.EjectCoreImportPath))
	mustCopyFile(destBinaryPath, app.BinaryPath)
	mustChmod(destBinaryPath, 0755)
	mustCopyDir(path.Join(tmpEjectPath, "conf"), path.Join(eject.EjectPath, "conf"), nil)
	mustCopyDir(path.Join(tmpEjectPath, "views"), path.Join(eject.EjectPath, "views"), nil)
	mustCopyDir(path.Join(srcPath, filepath.FromSlash(appImportPath)), eject.BasePath, nil)

	tmplData, runShPath := map[string]interface{}{
		"BinName":    filepath.Base(app.BinaryPath),
		"ImportPath": appImportPath,
		"Mode":       mode,
	}, path.Join(destPath, "run.sh")

	mustRenderTemplate(
		runShPath,
		filepath.Join(eject.EjectPath, "..", "cmd", "eject", "package_run.sh.template"),
		tmplData)

	mustChmod(runShPath, 0755)

	mustRenderTemplate(
		filepath.Join(destPath, "run.bat"),
		filepath.Join(eject.EjectPath, "..", "cmd", "eject", "package_run.bat.template"),
		tmplData)
}