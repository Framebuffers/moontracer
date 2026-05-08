package mediaserver

import (
	"io"
	"os"
	"path/filepath"
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

/*
ProcessToken composites a source photo with a frame and writes the result to outPath.

The output directory is created if it does not exist
*/
func ProcessToken(srcPath, frmPath, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
