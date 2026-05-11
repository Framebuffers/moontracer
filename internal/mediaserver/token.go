package mediaserver

import (
	"image/color"
	"os"
	"path/filepath"

	"moontracer/internal/tokengenerator"
)

/*
TokenPath generates disk and public URL paths for a player token file.

suffix distinguishes src/frm/out variants (e.g. "source", "frame", "out").
*/
func TokenPath(dataDir, baseURL, guildID, playerID, suffix, ext string) (disk, url string) {
	rel := filepath.Join(guildID, "tokens", playerID, suffix+ext)
	disk = filepath.Join(dataDir, rel)
	url = baseURL + "/" + filepath.ToSlash(rel)
	return
}

// ProcessToken composites a source photo with a frame image and writes the result to outPath.
func ProcessToken(srcPath, frmPath, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	_, err := tokengenerator.New(srcPath, frmPath, outPath)
	return err
}

// ProcessBasicToken composites a source photo with a solid-color gradient ring and writes the result to outPath.
func ProcessBasicToken(srcPath, outPath string, frameColor color.RGBA) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	_, err := tokengenerator.NewBasicToken(srcPath, frameColor, 32, outPath)
	return err
}
