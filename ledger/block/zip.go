package block


import (
"archive/zip"
"fmt"
"io"
"log"
"os"
"path/filepath"
"strings"
)

//func zipfile() {
//
//	var src = "log"
//
//	var dst = "log.zip"
//
//	if err := Zip(dst, src); err != nil {
//		log.Fatalln(err)
//	}
//}

func (df *TopiaFile)Zip(src string) (err error) {
	dst := df.File.Name()
	fw, err := os.Create(dst)
	defer fw.Close()
	if err != nil {
		return err
	}


	zw := zip.NewWriter(fw)
	defer func() {

		if err := zw.Close(); err != nil {
			log.Fatalln(err)
		}
	}()


	return filepath.Walk(src, func(path string, fi os.FileInfo, errBack error) (err error) {
		if errBack != nil {
			return errBack
		}


		fh, err := zip.FileInfoHeader(fi)
		if err != nil {
			return
		}


		fh.Name = strings.TrimPrefix(path, string(filepath.Separator))


		if fi.IsDir() {
			fh.Name += "/"
		}


		w, err := zw.CreateHeader(fh)
		if err != nil {
			return
		}

		if !fh.Mode().IsRegular() {
			return nil
		}


		fr, err := os.Open(path)
		defer fr.Close()
		if err != nil {
			return
		}


		n, err := io.Copy(w, fr)
		if err != nil {
			return
		}
		fmt.Println(n)

		return nil
	})
}