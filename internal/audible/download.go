package audible

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Download fetches the encrypted payload from the CDN.
//
// It streams the body to dest, reporting the number of bytes written
// through onProgress. The User-Agent is the one Audible's CDN expects.
//
// Success is reported when bytesWritten >= sourceSize.
func (a *Api) Download(ctx context.Context, url string, dest string, sourceSize int64, onProgress func(int64)) error {
	if err := a.ensureProfile(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Audible/671 CFNetwork/1240.0.4 Darwin/20.6.0")
	req.Header.Set("Accept", "*/*")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("audible: HTTP %d: %s", resp.StatusCode, string(data))
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("audible: cannot create %s: %w", dest, err)
	}
	defer f.Close()

	// Stream through two buffered writers, matching the original's
	// double-buffered copy.
	rdr := bufio.NewReaderSize(resp.Body, 16384)
	wrtr := bufio.NewWriterSize(f, 16384)

	buf := make([]byte, 16384)
	var written int64
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n, rerr := rdr.Read(buf)
		if n > 0 {
			nn, werr := wrtr.Write(buf[:n])
			if werr != nil {
				return fmt.Errorf("audible: cannot write %s: %w", dest, werr)
			}
			written += int64(nn)
			if onProgress != nil {
				onProgress(written)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fmt.Errorf("audible: cannot read the payload: %w", rerr)
		}
	}

	if err := wrtr.Flush(); err != nil {
		return fmt.Errorf("audible: cannot flush %s: %w", dest, err)
	}

	if sourceSize > 0 && written < sourceSize {
		return fmt.Errorf("audible: short download: %d of %d bytes", written, sourceSize)
	}

	return nil
}
