// package ucs provides Universal Category System (UCS) category and filename utilities.
package ucs

import (
	"embed"
	"encoding/csv"
	"io/fs"
	"os"
	"slices"
	"strings"
)

//go:embed *.csv
var content embed.FS

func open() (fs.File, error) {
	if fp := os.Getenv("UCS_CSV_FILE"); fp != "" {
		return os.Open(fp)
	}
	return content.Open("UCS-v8.2.csv")
}

// Category is UCS category.
type Category struct {
	Category    string
	SubCategory string
	CatID       string
	CatShort    string
	Synonyms    string
}

// Categories returns the full list of UCS categories.
//
// The builtin CSV file is used as a datasource unless UCS_CSV_FILE is set, in which case that file
// will be used instead. Compatible CSV files are availble at https://universalcategorysystem.com.
func Categories() ([]Category, error) {
	f, err := open()
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var list []Category
	for _, r := range records {
		if len(r) != 6 {
			continue
		}
		list = append(list, Category{
			Category:    r[0],
			SubCategory: r[1],
			CatID:       r[2],
			CatShort:    r[3],
			Synonyms:    r[5],
		})
	}
	slices.SortFunc(list, func(a, b Category) int {
		if a.CatID < b.CatID {
			return -1
		}
		return 1
	})
	return list, nil
}

// Filename is a UCS filename. Individual segments *must not* contain underscores, because
// underscores are used to separate segments in the rendered filename.
type Filename struct {
	CatID     string
	FXName    string
	CreatorID string
	SourceID  string
	UserData  string
}

// Render returns the assembled filename with the given extension:
//
//	CatID_FXName_CreatorID_SourceID_UserData.Extention
func (f Filename) Render(ext string) string {
	segs := []string{f.CatID, f.FXName, f.CreatorID, f.SourceID}
	if f.UserData != "" {
		segs = append(segs, f.UserData)
	}
	return strings.Join(segs, "_") + ext
}
