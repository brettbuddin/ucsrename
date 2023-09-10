// package renamer provides an interactive file renamer.
package renamer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/brettbuddin/ucsrename/ucs"
)

func NewDefault() (Renamer, error) {
	fzfExec, err := exec.LookPath("fzf")
	if err != nil {
		return Renamer{}, err
	}

	return Renamer{
		SelfCommand: os.Args[0],
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		FZFExec:     fzfExec,
	}, nil
}

// Renamer is an interactive renamer for UCS filenames.
type Renamer struct {
	SelfCommand string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	FZFExec     string
}

// Run executes a rename for the given file. It prompts the user for CatID, FXName, CreatorID,
// SourceID and UserData. A final confirmation is required unless forceConfirm is true.
func (r Renamer) Run(filename string, forceConfirm bool) error {
	srcFileInfo, err := os.Stat(filename)
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

	f, err := r.buildFilename()
	if err != nil {
		return err
	}
	newName := f.Render(ext)

	oldName := filepath.Base(srcFileInfo.Name())
	if forceConfirm {
		return os.Rename(oldName, newName)
	}

	return r.confirm(
		fmt.Sprintf("Rename %q to %q?", oldName, newName),
		func() error {
			return os.Rename(oldName, newName)
		},
	)
}

func (r Renamer) buildFilename() (ucs.Filename, error) {
	if catID := os.Getenv("UCS_CAT_ID"); catID != "" {
		if err := validateCatID(catID); err != nil {
			return ucs.Filename{}, err
		}
		return r.promptFields(catID)
	}

	cmd := exec.Command(
		r.FZFExec,
		"--ansi",
		"--no-preview",
		"--header=\nSelect a CatID",
	)
	var out bytes.Buffer
	cmd.Stdin = r.Stdin
	cmd.Stderr = r.Stderr
	cmd.Stdout = &out

	cmd.Env = append(os.Environ(), fmt.Sprintf("FZF_DEFAULT_COMMAND=%s", r.SelfCommand))
	if err := cmd.Run(); err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return ucs.Filename{}, err
		}
	}

	choice := strings.TrimSpace(out.String())
	choiceSegs := strings.Split(choice, " ")
	catID := strings.TrimRight(choiceSegs[0], ":")

	return r.promptFields(catID)
}

func (r Renamer) promptFields(catID string) (ucs.Filename, error) {
	f := ucs.Filename{
		CatID: catID,
	}

	fmt.Fprintf(r.Stdout, "CatID: %s\n", catID)

	var err error
	f.FXName, err = r.promptField("FXName", required, "")
	if err != nil {
		return f, err
	}
	if f.FXName == "" {
		return f, fmt.Errorf("FXName is required")
	}

	f.CreatorID, err = r.promptField("CreatorID", required, "UCS_CREATOR_ID")
	if err != nil {
		return f, err
	}
	if f.CreatorID == "" {
		return f, fmt.Errorf("CreatorID is required")
	}

	f.SourceID, err = r.promptField("SourceID", required, "UCS_SOURCE_ID")
	if err != nil {
		return f, err
	}
	if f.SourceID == "" {
		return f, fmt.Errorf("SourceID is required")
	}

	f.UserData, err = r.promptField("UserData", optional, "UCS_USER_DATA")
	if err != nil {
		return f, err
	}

	return f, nil
}

type requirement int

const (
	required requirement = iota
	optional
)

func (r Renamer) promptField(fieldName string, req requirement, envOverrideVar string) (string, error) {
	if envOverrideVar != "" {
		val := os.Getenv(envOverrideVar)
		if val != "" {
			return val, nil
		}
	}

	for {
		fmt.Fprintf(r.Stdout, "%s: ", fieldName)
		reader := bufio.NewReader(r.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		trimmed := strings.TrimSpace(text)
		if req == required && trimmed == "" {
			fmt.Fprintf(r.Stderr, "Invalid: %s is required\n", fieldName)
			continue
		}
		if strings.Contains(trimmed, "_") {
			fmt.Fprintln(r.Stderr, "Invalid: value cannot contain \"_\", because it is the filename field delimiter")
			continue
		}
		return strings.Join(strings.Fields(trimmed), "-"), nil
	}
}

func (r Renamer) confirm(prompt string, yes func() error) error {
	for {
		var confirm string
		fmt.Printf("%s (y/n) ", prompt)
		fmt.Fscanf(r.Stdin, "%s", &confirm)
		switch strings.ToLower(confirm) {
		case "y", "yes":
			return yes()
		case "n", "no":
			return nil
		default:
			return nil
		}
	}
}

func validateCatID(catID string) error {
	categories, err := ucs.Categories()
	if err != nil {
		return err
	}
	exists := slices.ContainsFunc(categories, func(c ucs.Category) bool {
		return c.CatID == catID
	})
	if !exists {
		return fmt.Errorf("unknown CatID: %s", catID)
	}
	return nil
}
