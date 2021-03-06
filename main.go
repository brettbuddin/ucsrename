package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
)

//go:embed UCS-v8.1.csv
var content embed.FS

func main() {
	if err := run(); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(2)
		}
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	if isInteractive(os.Stdout) {
		return runInteractive()
	}
	return printCategories()
}

func printCategories() error {
	f, err := content.Open("UCS-v8.1.csv")
	if err != nil {
		return err
	}

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	var items []ucsItem
	for _, r := range records {
		if len(r) != 6 {
			continue
		}
		items = append(items, ucsItem{
			Category:    r[0],
			SubCategory: r[1],
			CatID:       r[2],
			CatShort:    r[3],
			Synonyms:    r[5],
		})
	}

	for _, it := range items {
		fmt.Println(it.CatID, "|", it.Category, it.SubCategory, "|", it.Synonyms)
	}
	return nil
}

type ucsItem struct {
	Category    string
	SubCategory string
	CatID       string
	CatShort    string
	Synonyms    string
}

func runInteractive() error {
	var forceConfirm bool
	fs := flag.NewFlagSet("ucsrename", flag.ContinueOnError)
	fs.BoolVar(&forceConfirm, "y", false, "force confirm rename")
	fs.Usage = usageFn(fs)
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	srcFileInfo, err := os.Stat(fs.Arg(0))
	if err != nil {
		return err
	}
	if srcFileInfo.IsDir() {
		return fmt.Errorf("%s is a directory", srcFileInfo.Name())
	}
	ext := filepath.Ext(srcFileInfo.Name())
	if ext == "" {
		return fmt.Errorf("no file name extension found")
	}

	f, err := newUCSFilename()
	if err != nil {
		return err
	}
	newName := f.render(ext)

	oldName := filepath.Base(srcFileInfo.Name())
	if forceConfirm {
		return os.Rename(oldName, newName)
	}

	for {
		var confirm string
		fmt.Printf("Rename %q to %q? (y/n) ", oldName, newName)
		fmt.Scanf("%s", &confirm)
		switch strings.ToLower(confirm) {
		case "y", "yes":
			return os.Rename(oldName, newName)
		case "n", "no":
			return nil
		default:
		}
	}

	return nil
}

func fzfInstalled() bool {
	v, _ := exec.LookPath("fzf")
	if v != "" {
		return true
	}
	return false
}

func isInteractive(stdout *os.File) bool {
	return isatty.IsTerminal(stdout.Fd()) && fzfInstalled()
}

func newUCSFilename() (ucsFilename, error) {
	cmd := exec.Command("fzf", "--ansi", "--no-preview")
	var out bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = &out

	cmd.Env = append(os.Environ(), fmt.Sprintf("FZF_DEFAULT_COMMAND=%s", os.Args[0]))
	if err := cmd.Run(); err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return ucsFilename{}, err
		}
	}

	choice := strings.TrimSpace(out.String())
	choiceSegs := strings.Split(choice, " ")
	catID := choiceSegs[0]

	return promptFields(os.Stdin, catID)
}

type ucsFilename struct {
	catID     string
	fxName    string
	creatorID string
	sourceID  string
	userData  string
}

func promptFields(r io.Reader, catID string) (ucsFilename, error) {
	f := ucsFilename{
		catID: catID,
	}

	var err error
	f.fxName, err = prompt(os.Stdin, "FX Name", "")
	if err != nil {
		return f, err
	}
	if f.fxName == "" {
		return f, fmt.Errorf("FXName is required.")
	}

	f.creatorID, err = prompt(os.Stdin, "Creator ID", "UCS_CREATOR_ID")
	if err != nil {
		return f, err
	}
	if f.creatorID == "" {
		return f, fmt.Errorf("CreatorID is required.")
	}

	f.sourceID, err = prompt(os.Stdin, "Source ID", "UCS_SOURCE_ID")
	if err != nil {
		return f, err
	}
	if f.sourceID == "" {
		return f, fmt.Errorf("SourceID is required.")
	}

	f.userData, err = prompt(os.Stdin, "User Data (Optional)", "UCS_USER_DATA")
	if err != nil {
		return f, err
	}

	return f, nil
}

func (f ucsFilename) render(ext string) string {
	segs := []string{f.catID, f.fxName, f.creatorID, f.sourceID}
	if f.userData != "" {
		segs = append(segs, f.userData)
	}
	return strings.Join(segs, "_") + ext
}

func prompt(r io.Reader, fieldName string, defaultEnv string) (string, error) {
	if defaultEnv != "" {
		defaultValue := os.Getenv(defaultEnv)
		if defaultValue != "" {
			return defaultValue, nil
		}
	}

	fmt.Printf("%s: ", fieldName)
	reader := bufio.NewReader(r)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

var usage = `
ucsrename renames files using Universal Category System (UCS) filename pattern. 

Usage:
	
	ucsrename [-y] filename.wav

The program asks a series of questions to build a filename that conforms to UCS
standards. The source file's file extension is carried forward to the new file.
Here's the layout of the filename that it produces:

	CatID_FXName_CreatorID_SourceID_UserData.Extention

CatID, FXName, CreatorID and SourceID are required fields. The UserData field is
optional and can be to specify information not captured by the UCS standard.

fzf is required to provide a helpful, filterable, list of category IDs.

The UCS project has a great video outlining the filename structure:
https://www.youtube.com/watch?v=0s3ioIbNXSM
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
