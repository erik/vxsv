# vxsv

view x separated values.

A terminal viewer for tabular data (CSV, TSV, etc.) Can also be used
as a pager for scrolling through Postgres / MySQL command line output.

[![asciicast](https://i.imgur.com/HHKznJk.png)](https://asciinema.org/a/109283)

## installation

```bash
go get -u github.com/erik/vxsv/cmd/vxsv
```

## usage

```
$ vxsv --help

Usage:
  vxsv [--psql | --mysql | --delimiter=DELIM | --tabs]
       [--no-headers] [--count=N] [PATH | -]
  vxsv -h | --help

Arguments:
  PATH     file to load [defaults to stdin]

Options:
  -h --help                 show this help message and exit.
  -p --psql                 parse output of psql cli (used as a pager)
  -m --mysql                parse output of mysql cli
  -n --count=N              only read N records.
  -H --no-headers           don't read headers from first row (for separated values)
  -d --delimiter=DELIM      separator for values [default: ,].
  -t --tabs                 use tabs as separator value.
```

### postgres

```
$ PAGER='vxsv -p' psql ...
```

### mysql

```
$ mysql ...

mysql> \P vxsv -m
```
