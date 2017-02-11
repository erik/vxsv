package main

import (
	"fmt"
	"io"
	"os"

	"github.com/docopt/docopt-go"
)

type TabularData struct {
	Columns []string
	Rows    [][]string
}

func main() {
	usage := `view [x] separated values

Usage:
  vxsv [-tpxsd DELIMITER] ([-] | [<PATH>])

Arguments:
  PATH  file to load [default: -]

Options:
  -h --help                   show this help message and exit.
  -s --stream                 handle streaming data.
  -d --delimiter=<DELIMITER>  separator for values [default: ,].
  -t --tabs                   use tabs as separator value.
  -p --psql                   parse output of psql tool, when used as a pager.
`

	args, _ := docopt.Parse(usage, nil, true, "0.0.0", false)

	var reader io.Reader
	var data TabularData

	// default to stdin if we don't have an explicit file passed in
	if args["<PATH>"] != nil {
		file_name, _ := args["<PATH>"].(string)
		file, err := os.Open(file_name)
		if err != nil {
			panic("Failed to open file")
		}

		reader = io.Reader(file)
	} else {
		reader = io.Reader(os.Stdin)
	}

	if args["--psql"] == true {
		data = ReadPsqlTable(reader)
	}

	fmt.Println(args)
	UiLoop(data)
}
