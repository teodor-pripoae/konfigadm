package phases

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	. "github.com/moshloop/konfigadm/pkg/types"
)

var Packages AllPhases = packages{}

type packages struct{}

func (p packages) ApplyPhase(sys *Config, ctx *SystemContext) ([]Command, Filesystem, error) {
	commands := Commands{}
	files := Filesystem{}

	for _, repo := range *sys.PackageRepos {
		var os OS
		var err error
		if os, err = GetOSForTag(repo.Flags...); err != nil {
			return nil, nil, err
		}

		log.Tracef("Adding %s\n", repo)
		if repo.URL != "" {
			_commands := os.GetPackageManager().
				AddRepo(repo.URL, repo.Channel, repo.VersionCodeName, repo.Name, repo.GPGKey)
			commands.Append(_commands.WithTags(repo.Flags...))
		}
	}
	addPackageCommands(sys, &commands)
	_commands := commands.Merge()
	return _commands, files, nil
}

type packageOperations struct {
	install   []string
	uninstall []string
	mark      []string
	tags      []Flag
}

func appendStrings(slice []string, s string) []string {
	var newSlice []string
	if slice != nil {
		newSlice = slice
	}
	newSlice = append(newSlice, s)
	return newSlice
}

func getKeyFromTags(tags ...Flag) string {
	return fmt.Sprintf("%s", tags)
}

func addPackageCommands(sys *Config, commands *Commands) {
	var managers = make(map[string]packageOperations)

	for _, p := range *sys.Packages {
		if len(p.Flags) == 0 {
			continue
		}
		var ops packageOperations
		var ok bool
		if ops, ok = managers[getKeyFromTags(p.Flags...)]; !ok {
			ops = packageOperations{tags: p.Flags}

		}
		if p.Uninstall {
			ops.uninstall = appendStrings(ops.uninstall, p.Name)
		} else {
			ops.install = appendStrings(ops.install, p.Name)
		}
		managers[getKeyFromTags(p.Flags...)] = ops
	}

	for _, os := range BaseOperatingSystems {
		for _, p := range *sys.Packages {
			if len(p.Flags) > 0 {
				continue
			}
			var ops packageOperations
			var ok bool
			if ops, ok = managers[getKeyFromTags(os.GetTags()...)]; !ok {
				ops = packageOperations{tags: os.GetTags()}

			}
			if p.Uninstall {
				ops.uninstall = appendStrings(ops.uninstall, p.Name)
			} else {
				ops.install = appendStrings(ops.install, p.Name)
			}
			managers[getKeyFromTags(os.GetTags()...)] = ops
		}
	}

	for _, ops := range managers {
		os, _ := GetOSForTag(ops.tags...)
		commands.Append(os.GetPackageManager().Update().WithTags(ops.tags...))

		if ops.install != nil && len(ops.install) > 0 {
			commands = commands.Append(os.GetPackageManager().Install(ops.install...).WithTags(ops.tags...))
		}
		if ops.uninstall != nil && len(ops.uninstall) > 0 {
			commands = commands.Append(os.GetPackageManager().Uninstall(ops.uninstall...).WithTags(ops.tags...))
		}
		if ops.mark != nil && len(ops.mark) > 0 {
			commands = commands.Append(os.GetPackageManager().Mark(ops.mark...).WithTags(ops.tags...))
		}
	}

}

func (p packages) ProcessFlags(sys *Config, flags ...Flag) {
	minified := []Package{}
	for _, pkg := range *sys.Packages {
		if MatchAll(flags, pkg.Flags) {
			minified = append(minified, pkg)
		}
	}
	sys.Packages = &minified

	minifiedRepos := []PackageRepo{}
	for _, repo := range *sys.PackageRepos {
		if MatchesAny(flags, repo.Flags) {
			minifiedRepos = append(minifiedRepos, repo)
		}
	}
	sys.PackageRepos = &minifiedRepos
}

func (p packages) Verify(cfg *Config, results *VerifyResults, flags ...Flag) bool {
	verify := true
	var os OS
	var err error
	if os, err = GetOSForTag(flags...); err != nil {
		results.Fail("Unable to find OS for tags %s", flags)
		return false
	}
	for _, p := range *cfg.Packages {
		installed := os.GetPackageManager().GetInstalledVersion(p.Name)
		if p.Uninstall {
			if installed == "" {
				results.Pass("%s is not installed", p)
			} else {
				results.Fail("%s-%s should not be installed", p, installed)
				verify = false
			}
		} else if p.Version == "" && installed != "" {
			results.Pass("%s-%s is installed", p, installed)
		} else if p.Version == "" && installed == "" {
			results.Fail("%s is not installed, any version required", p)
			verify = false
		} else if installed == p.Version {
			results.Pass("%s-%s is installed", p, installed)
		} else {
			results.Fail("%s-%s is installed, but not the correct version: %s", p.Name, installed, p.Version)
			verify = false
		}
	}

	return verify
}