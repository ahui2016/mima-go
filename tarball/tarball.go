package tarball

// 参考 https://gist.github.com/maximilien/328c9ac19ab0a158a8df

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"

	"github.com/ahui2016/mima-go/util"
)

// CreateTarball 把 filePaths 里的全部文件打包压缩, 新建文件 tarballFilePath.
func CreateTarball(tarballFilePath string, filePaths []string) error {
	file, err := os.Create(tarballFilePath)
	if err != nil {
		return err
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)

	var allErrors []error
	for _, filePath := range filePaths {
		if err := addFileToTar(filePath, tarWriter); err != nil {
			allErrors = append(allErrors, err)
			break
		}
	}
	allErrors = append(allErrors, tarWriter.Close(), gzipWriter.Close(), file.Close())
	return util.WrapErrors(allErrors...)
}

// 把数据库文件以及碎片文件备份到一个 tar 文件里.
// 主要在 Rebuild 之前使用, 以防万一 rebuild 出错.
func (db *MimaDB) backupToTar() {
	pattern := filepath.Join(dbDirPath, "*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, file := range files {
		fileName := filepath.Base(file)
		fileBody := readFile(file)
		hdr := &tar.Header{
			Name: fileName,
			Mode: 0600,
			Size: int64(len(fileBody)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			panic(err)
		}
		if _, err := tw.Write(fileBody); err != nil {
			panic(err)
		}
	}
	if err := tw.Close(); err != nil {
		panic(err)
	}
}
