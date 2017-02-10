package main

import "github.com/docopt/docopt-go"
import "fmt"

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

	args, _ := docopt.Parse(usage, nil, true, "v0", false)

	// default to stdin if we don't have an explicit file passed in
	if args["<PATH>"] == nil {
		args["<PATH>"] = "-"
	}

	if args["--psql"] == true {

	}

	fmt.Println(args)
	UiLoop()
}
