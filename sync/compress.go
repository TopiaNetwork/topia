package sync

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"io"
)

type CompressType byte

const (
	ZIP CompressType = iota
	GZIP
)

var ErrCompressType = errors.New("CompressType error, use ZIP or GZIP")

func ZipBytes(data []byte) ([]byte, error) {
	var in bytes.Buffer
	z := zlib.NewWriter(&in)
	_, err := z.Write(data)
	if err != nil {
		return nil, err
	}
	err = z.Close()
	if err != nil {
		return nil, err
	}
	return in.Bytes(), nil
}
func UnZipBytes(data []byte) ([]byte, error) {
	var out bytes.Buffer
	var in bytes.Buffer
	in.Write(data)
	r, err := zlib.NewReader(&in)
	if err != nil {
		return nil, err
	}
	err = r.Close()
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(&out, r)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
func GzipBytes(data []byte) ([]byte, error) {
	var bufOut bytes.Buffer
	gw := gzip.NewWriter(&bufOut)
	_, err := gw.Write(data)
	if err != nil {
		return nil, err
	}
	err = gw.Close()
	if err != nil {
		return nil, err
	}
	return bufOut.Bytes(), nil
}

func UnGzipBytes(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	if err != nil {
		return nil, err
	}
	err = gz.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func CompressBytes(data []byte, ct CompressType) ([]byte, error) {
	switch ct {
	case ZIP:
		return ZipBytes(data)
	case GZIP:
		return GzipBytes(data)
	default:
		return nil, ErrCompressType
	}
}

func UnCompressBytes(data []byte, ct CompressType) ([]byte, error) {
	switch ct {
	case ZIP:
		return UnZipBytes(data)
	case GZIP:
		return UnGzipBytes(data)
	default:
		return nil, ErrCompressType
	}
}
