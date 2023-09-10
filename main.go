package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/brettbuddin/ucsrename/renamer"
	"github.com/brettbuddin/ucsrename/ucs"
	"github.com/mattn/go-isatty"
)

func main() {
	err := run()
	if err == nil {
		return
	}
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func run() error {
	if !isInteractive(os.Stdout) {
		return printCategories(os.Stdout)
	}

	var forceConfirm bool
	fs := flag.NewFlagSet("ucsrename", flag.ContinueOnError)
	fs.BoolVar(&forceConfirm, "y", false, "force confirm rename")
	fs.Usage = usageFn(fs)
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	filename := fs.Arg(0)
	if filename == "" {
		fs.Usage()
		return nil
	}

	fzfExec, err := exec.LookPath("fzf")
	if err != nil {
		return err
	}

	r := renamer.Renamer{
		SelfCommand: os.Args[0],
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		FZFExec:     fzfExec,
	}
	return r.Run(filename, forceConfirm)
}

func isInteractive(stdout *os.File) bool {
	return isatty.IsTerminal(stdout.Fd())
}

func printCategories(w io.Writer) error {
	categories, err := ucs.Categories()
	if err != nil {
		return err
	}

	for _, c := range categories {
		fmt.Fprintf(w, "%s: %s %s -- %s\n", c.CatID, c.Category, c.SubCategory, c.Synonyms)
	}
	return nil
}

var usage = `
ucsrename renames files using Universal Category System (UCS) filename pattern.

Usage:
	
	ucsrename [-y] filename.wav

The program asks a series of questions to build a filename that conforms to UCS standards. The
source file's file extension is carried forward to the new file. Here's the layout of the filename
that it produces:

	CatID_FXName_CreatorID_SourceID_UserData.Extention

CatID, FXName, CreatorID and SourceID are required fields. The UserData field is optional and can be
used to specify information not captured by the UCS standard.

The program will prompt you for these fields, but some fields can be skipped by setting one of the
following environment variables:

- UCS_CAT_ID
- UCS_CREATOR_ID
- UCS_SOURCE_ID
- UCS_USER_DATA

Once a variable is set in the environment, the program will use that value instead of prompting the
user. This is useful for relatively static fields like CreatorID and SourceID.

fzf is required to provide a helpful, filterable, list of category IDs.

The UCS project has a great video outlining the filename structure:
https://www.youtube.com/watch?v=0s3ioIbNXSM

A UCS CSV is embedded in the program, but that file can be overridden by setting UCS_CSV_FILE
environment variable. Once set, all invocations will use that file instead of the embedded UCS CSV
file.
`

func usageFn(fs *flag.FlagSet) func() {
	return func() {
		out := fs.Output()
		fmt.Fprintln(out, usage)
		fmt.Fprint(out, "Flags:\n\n")
		fs.PrintDefaults()
		fmt.Fprintln(out)
	}
}
