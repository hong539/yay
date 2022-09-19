package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Jguer/yay/v11/pkg/db"
	"github.com/Jguer/yay/v11/pkg/dep"
	"github.com/Jguer/yay/v11/pkg/multierror"
	"github.com/Jguer/yay/v11/pkg/settings"
	"github.com/Jguer/yay/v11/pkg/settings/parser"
	"github.com/Jguer/yay/v11/pkg/text"
	"github.com/leonelquinteros/gotext"

	mapset "github.com/deckarep/golang-set/v2"
)

type (
	PostInstallHookFunc func(ctx context.Context) error
	Installer           struct {
		dbExecutor       db.Executor
		postInstallHooks []PostInstallHookFunc
	}
)

func (installer *Installer) AddPostInstallHook(hook PostInstallHookFunc) {
	installer.postInstallHooks = append(installer.postInstallHooks, hook)
}

func (installer *Installer) RunPostInstallHooks(ctx context.Context) error {
	var errMulti multierror.MultiError

	for _, hook := range installer.postInstallHooks {
		if err := hook(ctx); err != nil {
			errMulti.Add(err)
		}
	}
	return errMulti.Return()
}

func (installer *Installer) Install(ctx context.Context,
	cmdArgs *parser.Arguments,
	targets []map[string]*dep.InstallInfo,
	pkgBuildDirs map[string]string,
) error {
	// Reorganize targets into layers of dependencies
	for i := len(targets) - 1; i >= 0; i-- {
		err := installer.handleLayer(ctx, cmdArgs, targets[i], pkgBuildDirs)
		if err != nil {
			// rollback
			return err
		}
	}

	return nil
}

func (installer *Installer) handleLayer(ctx context.Context,
	cmdArgs *parser.Arguments, layer map[string]*dep.InstallInfo, pkgBuildDirs map[string]string,
) error {
	// Install layer
	nameToBaseMap := make(map[string]string, 0)
	syncDeps, syncExp := mapset.NewThreadUnsafeSet[string](), mapset.NewThreadUnsafeSet[string]()
	aurDeps, aurExp := mapset.NewThreadUnsafeSet[string](), mapset.NewThreadUnsafeSet[string]()

	for name, info := range layer {
		switch info.Source {
		case dep.AUR, dep.SrcInfo:
			nameToBaseMap[name] = *info.AURBase

			switch info.Reason {
			case dep.Explicit:
				if cmdArgs.ExistsArg("asdeps", "asdep") {
					aurDeps.Add(name)
				} else {
					aurExp.Add(name)
				}
			case dep.Dep, dep.MakeDep, dep.CheckDep:
				aurDeps.Add(name)
			}
		case dep.Sync:
			compositePkgName := fmt.Sprintf("%s/%s", *info.SyncDBName, name)

			switch info.Reason {
			case dep.Explicit:
				if cmdArgs.ExistsArg("asdeps", "asdep") {
					syncDeps.Add(compositePkgName)
				} else {
					syncExp.Add(compositePkgName)
				}
			case dep.Dep, dep.MakeDep, dep.CheckDep:
				syncDeps.Add(compositePkgName)
			}
		}
	}

	text.Debugln("syncDeps", syncDeps, "SyncExp", syncExp, "aurDeps", aurDeps, "aurExp", aurExp)

	errShow := installer.installSyncPackages(ctx, cmdArgs, syncDeps, syncExp)
	if errShow != nil {
		return ErrInstallRepoPkgs
	}

	errAur := installer.installAURPackages(ctx, cmdArgs, aurDeps, aurExp, nameToBaseMap, pkgBuildDirs, false)

	return errAur
}

func (installer *Installer) installAURPackages(ctx context.Context,
	cmdArgs *parser.Arguments,
	aurDepNames, aurExpNames mapset.Set[string],
	nameToBase, pkgBuildDirsByBase map[string]string,
	installIncompatible bool,
) error {
	deps, exp := make([]string, 0, aurDepNames.Cardinality()), make([]string, 0, aurExpNames.Cardinality())

	for _, name := range aurDepNames.Union(aurExpNames).ToSlice() {
		base := nameToBase[name]
		dir := pkgBuildDirsByBase[base]
		args := []string{"--nobuild", "-fC"}

		if installIncompatible {
			args = append(args, "--ignorearch")
		}

		// pkgver bump
		if err := config.Runtime.CmdBuilder.Show(
			config.Runtime.CmdBuilder.BuildMakepkgCmd(ctx, dir, args...)); err != nil {
			return fmt.Errorf("%s - %w", gotext.Get("error making: %s", base), err)
		}

		pkgdests, _, errList := parsePackageList(ctx, dir)
		if errList != nil {
			return errList
		}

		args = []string{"-cf", "--noconfirm", "--noextract", "--noprepare", "--holdver"}

		if installIncompatible {
			args = append(args, "--ignorearch")
		}

		if errMake := config.Runtime.CmdBuilder.Show(
			config.Runtime.CmdBuilder.BuildMakepkgCmd(ctx,
				dir, args...)); errMake != nil {
			return fmt.Errorf("%s - %w", gotext.Get("error making: %s", base), errMake)
		}

		names, err := installer.getNewTargets(pkgdests, name)
		if err != nil {
			return err
		}

		isDep := installer.isDep(cmdArgs, aurExpNames, name)

		if isDep {
			deps = append(deps, names...)
		} else {
			exp = append(exp, names...)
		}
	}

	if err := doInstall(ctx, cmdArgs, deps, exp); err != nil {
		return fmt.Errorf("%s - %w", fmt.Sprintf(gotext.Get("error installing:")+" %v %v", deps, exp), err)
	}

	return nil
}

func (*Installer) isDep(cmdArgs *parser.Arguments, aurExpNames mapset.Set[string], name string) bool {
	switch {
	case cmdArgs.ExistsArg("asdeps", "asdep"):
		return true
	case cmdArgs.ExistsArg("asexplicit", "asexp"):
		return false
	case aurExpNames.Contains(name):
		return false
	}

	return true
}

func (installer *Installer) getNewTargets(pkgdests map[string]string, name string,
) ([]string, error) {
	pkgdest, ok := pkgdests[name]
	names := make([]string, 0, 2)
	if !ok {
		return nil, &PkgDestNotInListError{name: name}
	}

	if _, errStat := os.Stat(pkgdest); os.IsNotExist(errStat) {
		return nil, &UnableToFindPkgDestError{name: name, pkgDest: pkgdest}
	}

	names = append(names, name)

	debugName := pkgdest + "-debug"
	pkgdestDebug, ok := pkgdests[debugName]
	if ok {
		if _, errStat := os.Stat(pkgdestDebug); errStat == nil {
			names = append(names, debugName)
		}
	}

	return names, nil
}

func (*Installer) installSyncPackages(ctx context.Context, cmdArgs *parser.Arguments,
	syncDeps, // repo targets that are deps
	syncExp mapset.Set[string], // repo targets that are exp
) error {
	repoTargets := syncDeps.Union(syncExp).ToSlice()
	if len(repoTargets) == 0 {
		return nil
	}

	arguments := cmdArgs.Copy()
	arguments.DelArg("asdeps", "asdep")
	arguments.DelArg("asexplicit", "asexp")
	arguments.DelArg("i", "install")
	arguments.Op = "S"
	arguments.ClearTargets()
	arguments.AddTarget(repoTargets...)

	errShow := config.Runtime.CmdBuilder.Show(config.Runtime.CmdBuilder.BuildPacmanCmd(ctx,
		arguments, config.Runtime.Mode, settings.NoConfirm))

	if errD := asdeps(ctx, cmdArgs, syncDeps.ToSlice()); errD != nil {
		return errD
	}

	if errE := asexp(ctx, cmdArgs, syncExp.ToSlice()); errE != nil {
		return errE
	}
	return errShow
}
