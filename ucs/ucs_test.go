package ucs

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltinCategories(t *testing.T) {
	categories, err := Categories()
	require.NoError(t, err)
	require.NotEmpty(t, categories, "builtin file isn't empty")
	require.True(t, slices.IsSortedFunc(categories, func(a, b Category) int {
		if a.CatID < b.CatID {
			return -1
		}
		return 1
	}), "ascending order")

	// Spot-check the structure by looking at AMBPark
	ambParkIndex := slices.IndexFunc(categories, func(c Category) bool {
		return c.CatID == "AMBPark"
	})
	ambPark := categories[ambParkIndex]
	require.Equal(t, "AMB", ambPark.CatShort)
	require.Equal(t, "AMBIENCE", ambPark.Category)
	require.Equal(t, "PARK", ambPark.SubCategory)
	require.Contains(t, ambPark.Synonyms, "park")
}

func TestOverrideCategories(t *testing.T) {
	reset := setEnv("UCS_CSV_FILE", filepath.Join("testdata", "override.csv"))
	t.Cleanup(reset)

	categories, err := Categories()
	require.NoError(t, err)
	require.Len(t, categories, 1, "override file only has one entry")
	require.Equal(t, "AIRBlow", categories[0].CatID)
}

func setEnv(key, value string) func() {
	orig := os.Getenv(key)
	os.Setenv(key, value)
	return func() {
		os.Setenv(key, orig)
	}
}

func TestFilenameRendering(t *testing.T) {
	filename := Filename{
		CatID:     "AMBPark",
		FXName:    "Central Park Bethesda Fountain",
		CreatorID: "Buddin",
		SourceID:  "Phonogrifter",
		UserData:  "Clippy",
	}
	require.Equal(t, "AMBPark_Central Park Bethesda Fountain_Buddin_Phonogrifter_Clippy.wav", filename.Render(".wav"))
}
