package main

import (
	"analyzers/baninit"
	"analyzers/circular"
	"analyzers/crossimport"
	"analyzers/filecount"
	"analyzers/filesize"
	"analyzers/importlocation"
	"analyzers/ioseparation"
	"analyzers/mutablestate"
	"analyzers/reexport"
	"analyzers/sharedtypes"
	"analyzers/wiringonly"

	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		crossimport.Analyzer,
		sharedtypes.Analyzer,
		wiringonly.Analyzer,
		circular.Analyzer,
		mutablestate.Analyzer,
		baninit.Analyzer,
		importlocation.Analyzer,
		ioseparation.Analyzer,
		reexport.Analyzer,
		filesize.Analyzer,
		filecount.Analyzer,
	)
}
