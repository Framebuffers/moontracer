package mediaserver

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

/*
MaxDownloadBytes caps remote response bodies (16 MiB, double the upload limit
to absorb encoding overhead).

Anything larger is treated as a malicious or misconfigured upstream and rejected.
*/
const MaxDownloadBytes = 16 * 1024 * 1024

/*
CoverPath returns the disk path and URL path for a campaign cover asset.

	disk:   data/[guildID]/assets/campaigns/[campaignID]/cover/[uuid][ext]
	url:    [baseURL]/[guildID]/assets/campaigns/[campaignID]/cover/[uuid][ext]
*/
func CoverPath(dataDir, baseURL, guildID, campaignID, ext string) (diskPath, publicURL string) {
	filename := uuid.NewString() + ext
	rel := filepath.Join(guildID, "assets", "campaigns", campaignID, "cover", filename)
	diskPath = filepath.Join(dataDir, rel)
	publicURL = fmt.Sprintf("%s/%s/assets/campaigns/%s/cover/%s", baseURL, guildID, campaignID, filename)
	return
}

/*
Download fetches srcURL and writes it atomically to diskPath.
Creates any missing parent directories. Returns the detected MIME type.
*/
func Download(srcURL, diskPath string) (mimeType string, err error) {
	if err = os.MkdirAll(filepath.Dir(diskPath), 0755); err != nil {
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(srcURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// NOTE: Cap reads at MaxDownloadBytes+1 so an oversize body is detectable.
	limited := io.LimitReader(resp.Body, MaxDownloadBytes+1)

	// NOTE: Read first 512 bytes to detect MIME, then prepend them back.
	buf := make([]byte, 512)
	n, err := io.ReadFull(limited, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return
	}
	mimeType = http.DetectContentType(buf[:n])
	if !strings.HasPrefix(mimeType, "image/") {
		err = fmt.Errorf("download: rejected non-image content-type %q", mimeType)
		return
	}

	tmp, err := os.CreateTemp(filepath.Dir(diskPath), ".tmp-*")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		if err != nil {
			os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(buf[:n]); err != nil {
		return
	}
	var copied int64
	if copied, err = io.Copy(tmp, limited); err != nil {
		return
	}
	if int64(n)+copied > MaxDownloadBytes {
		err = fmt.Errorf("download: response exceeded %d bytes", MaxDownloadBytes)
		return
	}
	if err = tmp.Close(); err != nil {
		return
	}

	err = os.Rename(tmpName, diskPath)
	return
}
