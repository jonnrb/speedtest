package speedtest

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"

	"github.com/jonnrb/speedtest/prober"
)

const (
	concurrentUploadLimit = concurrentDownloadLimit
	uploadRepeats         = downloadRepeats * 25

	safeChars = "0123456789abcdefghijklmnopqrstuv"
)

var uploadSizes = []int{1000 * 1000 / 4, 1000 * 1000 / 2}

// Will probe upload speed until enough samples are taken or ctx expires.
func (s Server) ProbeUploadSpeed(ctx context.Context, client *Client, stream chan BytesPerSecond) (BytesPerSecond, error) {
	grp := prober.NewGroup(concurrentUploadLimit)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, size := range uploadSizes {
		for i := 0; i < uploadRepeats; i++ {
			grp.Add(func(size int) func() (prober.BytesTransferred, error) {
				return func() (prober.BytesTransferred, error) {
					err := client.uploadFile(ctx, s.URL, size)
					if err != nil {
						return prober.BytesTransferred(0), err
					} else {
						return prober.BytesTransferred(size), nil
					}
				}
			}(size))
		}
	}

	return speedCollect(grp, stream)
}

type safeReader struct {
	in io.Reader
}

func (r safeReader) Read(p []byte) (n int, err error) {
	n, err = r.in.Read(p)
	for i := 0; i < n; i++ {
		p[i] = safeChars[p[i]&31]
	}
	return n, err
}

func (c *Client) uploadFile(ctx context.Context, url string, size int) error {
	res, err := c.post(ctx, url, "application/x-www-form-urlencoded",
		io.MultiReader(
			strings.NewReader("content1="),
			io.LimitReader(&safeReader{rand.Reader}, int64(size-9))))
	if err != nil {
		return fmt.Errorf("upload to %q failed: %v", url, err)
	}
	defer res.Body.Close()

	return nil
}
