package utils

import (
	"archive/zip"

	"bytes"

	"io"

	"os"

	"path/filepath"
)

// Compress compresses the given file or directory and returns the compressed data.

func Compress(filePath string) ([]byte, error) {

	buf := new(bytes.Buffer)

	zipWriter := zip.NewWriter(buf)

	err := filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {

		if err != nil {

			return err

		}

		header, err := zip.FileInfoHeader(info)

		if err != nil {

			return err

		}

		header.Name, err = filepath.Rel(filepath.Dir(filePath), path)

		if err != nil {

			return err

		}

		if info.IsDir() {

			header.Name += "/"

		} else {

			header.Method = zip.Deflate

		}

		writer, err := zipWriter.CreateHeader(header)

		if err != nil {

			return err

		}

		if !info.IsDir() {

			file, err := os.Open(path)

			if err != nil {

				return err

			}

			defer file.Close()

			_, err = io.Copy(writer, file)

			if err != nil {

				return err

			}

		}

		return nil

	})

	if err != nil {

		return nil, err

	}

	err = zipWriter.Close()

	if err != nil {

		return nil, err

	}

	return buf.Bytes(), nil

}
