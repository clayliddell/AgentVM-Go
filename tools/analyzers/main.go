package main

import (
	"analyzers/baninit"
	"analyzers/circular"
	"analyzers/crossimport"
	"analyzers/filecount"
	"analyzers/filesize"
	"analyzers/importlocation"
	"analyzers/mutablestate"

	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		crossimport.Analyzer,
		circular.Analyzer,
		mutablestate.Analyzer,
		baninit.Analyzer,
		importlocation.Analyzer,
		filesize.Analyzer,
		filecount.Analyzer,
	)
}
