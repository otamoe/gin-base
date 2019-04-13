package size

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Reader struct {
	Remaining  int64
	ctx        *gin.Context
	rdr        io.ReadCloser
	wasAborted bool
	sawEOF     bool
}

func (mbr *Reader) tooLarge() (n int, err error) {
	err = errors.New(http.StatusText(http.StatusRequestEntityTooLarge))
	if !mbr.wasAborted {
		mbr.wasAborted = true
		ctx := mbr.ctx
		ctx.Header("connection", "close")
		ctx.Status(http.StatusRequestEntityTooLarge)
	}
	return
}

func (mbr *Reader) Read(p []byte) (n int, err error) {
	toRead := mbr.Remaining
	if mbr.Remaining == 0 {
		if mbr.sawEOF {
			return mbr.tooLarge()
		}
		// The underlying io.Reader may not return (0, io.EOF)
		// at EOF if the requested size is 0, so read 1 byte
		// instead. The io.Reader docs are a bit ambiguous
		// about the return value of Read when 0 bytes are
		// requested, and {bytes,strings}.Reader gets it wrong
		// too (it returns (0, nil) even at EOF).
		toRead = 1
	}
	if int64(len(p)) > toRead {
		p = p[:toRead]
	}
	n, err = mbr.rdr.Read(p)
	if err == io.EOF {
		mbr.sawEOF = true
	}
	if mbr.Remaining == 0 {
		// If we had zero bytes to read Remaining (but hadn't seen EOF)
		// and we get a byte here, that means we went over our limit.
		if n > 0 {
			return mbr.tooLarge()
		}
		return 0, err
	}
	mbr.Remaining -= int64(n)
	if mbr.Remaining < 0 {
		mbr.Remaining = 0
	}
	return
}

func (mbr *Reader) Close() error {
	return mbr.rdr.Close()
}

func Middleware(limit int64) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Request.Body = &Reader{
			ctx:       ctx,
			rdr:       ctx.Request.Body,
			Remaining: limit,
		}
		ctx.Next()
	}
}
