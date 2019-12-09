package main

import "github.com/mafei198/goslib/excel"

func main() {
	err := excel.ExportGoAndJSON(
		"./excels",
		"server",
		"./gen/json_files",
		"./gen/gd",
		"gd")
	if err != nil {
		panic(err)
	}
	err = excel.CreateMergeJSON("./gen/json_files", "./gen/configData.json.gz", true)
	if err != nil {
		panic(err)
	}
}
