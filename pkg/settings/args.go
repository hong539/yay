package settings

import (
	"strconv"
	"strings"

	"github.com/Jguer/yay/v12/pkg/settings/parser"
	"github.com/Jguer/yay/v12/pkg/text"
)

func (c *Configuration) ParseCommandLine(a *parser.Arguments) error {
	if err := a.Parse(); err != nil {
		return err
	}

	c.extractYayOptions(a)

	// Reload CmdBuilder
	c.Runtime.CmdBuilder = c.CmdBuilder(nil)

	return nil
}

func (c *Configuration) extractYayOptions(a *parser.Arguments) {
	for option, value := range a.Options {
		if c.handleOption(option, value.First()) {
			a.DelArg(option)
		}
	}

	c.AURURL = strings.TrimRight(c.AURURL, "/")

	// if AurRPCURL is set, use that for /rpc calls
	if c.AURRPCURL == "" {
		c.AURRPCURL = c.AURURL + "/rpc?"
		return
	}

	if !strings.HasSuffix(c.AURRPCURL, "?") {
		if strings.HasSuffix(c.AURRPCURL, "/rpc") {
			c.AURRPCURL += "?"
		} else {
			c.AURRPCURL = strings.TrimRight(c.AURRPCURL, "/") + "/rpc?"
		}
	}
}

func (c *Configuration) handleOption(option, value string) bool {
	switch option {
	case "aururl":
		c.AURURL = value
	case "aurrpcurl":
		c.AURRPCURL = value
	case "save":
		c.SaveConfig = true
	case "afterclean", "cleanafter":
		c.CleanAfter = true
	case "noafterclean", "nocleanafter":
		c.CleanAfter = false
	case "debug":
		c.Debug = true
		text.GlobalLogger.Debug = true
		return false
	case "devel":
		c.Devel = true
	case "nodevel":
		c.Devel = false
	case "timeupdate":
		c.TimeUpdate = true
	case "notimeupdate":
		c.TimeUpdate = false
	case "topdown":
		c.BottomUp = false
	case "bottomup":
		c.BottomUp = true
	case "singlelineresults":
		c.SingleLineResults = true
	case "doublelineresults":
		c.SingleLineResults = false
	case "completioninterval":
		n, err := strconv.Atoi(value)
		if err == nil {
			c.CompletionInterval = n
		}
	case "sortby":
		c.SortBy = value
	case "searchby":
		c.SearchBy = value
	case "noconfirm":
		NoConfirm = true
	case "config":
		c.PacmanConf = value
	case "redownload":
		c.ReDownload = "yes"
	case "redownloadall":
		c.ReDownload = "all"
	case "noredownload":
		c.ReDownload = "no"
	case "rebuild":
		c.ReBuild = parser.RebuildModeYes
	case "rebuildall":
		c.ReBuild = parser.RebuildModeAll
	case "rebuildtree":
		c.ReBuild = parser.RebuildModeTree
	case "norebuild":
		c.ReBuild = parser.RebuildModeNo
	case "batchinstall":
		c.BatchInstall = true
	case "nobatchinstall":
		c.BatchInstall = false
	case "answerclean":
		c.AnswerClean = value
	case "noanswerclean":
		c.AnswerClean = ""
	case "answerdiff":
		c.AnswerDiff = value
	case "noanswerdiff":
		c.AnswerDiff = ""
	case "answeredit":
		c.AnswerEdit = value
	case "noansweredit":
		c.AnswerEdit = ""
	case "answerupgrade":
		c.AnswerUpgrade = value
	case "noanswerupgrade":
		c.AnswerUpgrade = ""
	case "gpgflags":
		c.GpgFlags = value
	case "mflags":
		c.MFlags = value
	case "gitflags":
		c.GitFlags = value
	case "builddir":
		c.BuildDir = value
	case "editor":
		c.Editor = value
	case "editorflags":
		c.EditorFlags = value
	case "makepkg":
		c.MakepkgBin = value
	case "makepkgconf":
		c.MakepkgConf = value
	case "nomakepkgconf":
		c.MakepkgConf = ""
	case "pacman":
		c.PacmanBin = value
	case "git":
		c.GitBin = value
	case "gpg":
		c.GpgBin = value
	case "sudo":
		c.SudoBin = value
	case "sudoflags":
		c.SudoFlags = value
	case "requestsplitn":
		n, err := strconv.Atoi(value)
		if err == nil && n > 0 {
			c.RequestSplitN = n
		}
	case "sudoloop":
		c.SudoLoop = true
	case "nosudoloop":
		c.SudoLoop = false
	case "provides":
		c.Provides = true
	case "noprovides":
		c.Provides = false
	case "pgpfetch":
		c.PGPFetch = true
	case "nopgpfetch":
		c.PGPFetch = false
	case "cleanmenu":
		c.CleanMenu = true
	case "nocleanmenu":
		c.CleanMenu = false
	case "diffmenu":
		c.DiffMenu = true
	case "nodiffmenu":
		c.DiffMenu = false
	case "editmenu":
		c.EditMenu = true
	case "noeditmenu":
		c.EditMenu = false
	case "useask":
		c.UseAsk = true
	case "nouseask":
		c.UseAsk = false
	case "combinedupgrade":
		c.CombinedUpgrade = true
	case "nocombinedupgrade":
		c.CombinedUpgrade = false
	case "a", "aur":
		c.Mode = parser.ModeAUR
	case "repo":
		c.Mode = parser.ModeRepo
	case "removemake":
		c.RemoveMake = "yes"
	case "noremovemake":
		c.RemoveMake = "no"
	case "askremovemake":
		c.RemoveMake = "ask"
	case "askyesremovemake":
		c.RemoveMake = "askyes"
	case "separatesources":
		c.SeparateSources = true
	case "noseparatesources":
		c.SeparateSources = false
	default:
		return false
	}

	return true
}
