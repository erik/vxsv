package main

import (
	"fmt"
	"io"
	"os"

	"github.com/docopt/docopt-go"
)

type Column struct {
	Name  string
	Width int
	// TODO: type
}

type TabularData struct {
	Width   int
	Columns []Column
	Rows    [][]string
}

func main() {
	usage := `view [x] separated values

Usage:
  vxsv [-tpxsd DELIMITER] ([-] | [<PATH>])
  vxsv -h | --help

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

	var data *TabularData
	var err error

	// default to stdin if we don't have an explicit file passed in
	reader := io.Reader(os.Stdin)

	if args["<PATH>"] != nil {
		file_name, _ := args["<PATH>"].(string)
		file, err := os.Open(file_name)
		if err != nil {
			panic("Failed to open file")
		}

		reader = io.Reader(file)
	}

	if args["--psql"] == true {
		if data, err = ReadPsqlTable(reader); err != nil {
			fmt.Printf("Failed to read PSQL data: %v", err)
			os.Exit(1)
		}
	} else {
		delimiter := ','
		if args["-t"] == true {
			delimiter = '\t'
		} else if args["--delimiter"] != nil {
			if delimiterStr, ok := args["--delimiter"].(string); !ok {
				panic("Couldn't grab delimiter")
			} else {
				delimiter = []rune(delimiterStr)[0]
			}
		}

		if data, err = ReadCSVFile(reader, delimiter); err != nil {
			fmt.Printf("Failed to read CSV file (do you have the right delimiter?): %v\n", err)
			os.Exit(1)
		}
	}

	ui := NewUi(data)
	if err := ui.Init(); err != nil {
		fmt.Printf("Failed to initialize terminal UI: %v\n", err)
		os.Exit(1)
	}

	ui.Loop()
}
