/*
 * Minio Cloud Storage, (C) 2016 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"io"
	"os"
	pathutil "path"
)

// Removes only the file at given path does not remove
// any parent directories, handles long paths for
// windows automatically.
func fsRemoveFile(filePath string) (err error) {
	if filePath == "" {
		return errInvalidArgument
	}

	if err = checkPathLength(filePath); err != nil {
		return err
	}

	if err = os.Remove(preparePath(filePath)); err != nil {
		if os.IsNotExist(err) {
			return errFileNotFound
		} else if os.IsPermission(err) {
			return errFileAccessDenied
		}
		return err
	}

	return nil
}

// Removes all files and folders at a given path, handles
// long paths for windows automatically.
func fsRemoveAll(dirPath string) (err error) {
	if dirPath == "" {
		return errInvalidArgument
	}

	if err = checkPathLength(dirPath); err != nil {
		return err
	}

	if err = removeAll(dirPath); err != nil {
		if os.IsPermission(err) {
			return errVolumeAccessDenied
		}
	}

	return err

}

// Removes a directory only if its empty, handles long
// paths for windows automatically.
func fsRemoveDir(dirPath string) (err error) {
	if dirPath == "" {
		return errInvalidArgument
	}

	if err = checkPathLength(dirPath); err != nil {
		return err
	}

	if err = os.Remove(preparePath(dirPath)); err != nil {
		if os.IsNotExist(err) {
			return errVolumeNotFound
		} else if isSysErrNotEmpty(err) {
			return errVolumeNotEmpty
		}
	}

	return err
}

// Creates a new directory, parent dir should exist
// otherwise returns an error. If directory already
// exists returns an error. Windows long paths
// are handled automatically.
func fsMkdir(dirPath string) (err error) {
	if dirPath == "" {
		return errInvalidArgument
	}

	if err = checkPathLength(dirPath); err != nil {
		return err
	}

	if err = os.Mkdir(preparePath(dirPath), 0777); err != nil {
		if os.IsExist(err) {
			return errVolumeExists
		} else if os.IsPermission(err) {
			return errDiskAccessDenied
		} else if isSysErrNotDir(err) {
			// File path cannot be verified since
			// one of the parents is a file.
			return errDiskAccessDenied
		} else if isSysErrPathNotFound(err) {
			// Add specific case for windows.
			return errDiskAccessDenied
		}
	}

	return nil
}

// Lookup if directory exists, returns directory
// attributes upon success.
func fsStatDir(statDir string) (os.FileInfo, error) {
	if statDir == "" {
		return nil, errInvalidArgument
	}
	if err := checkPathLength(statDir); err != nil {
		return nil, err
	}

	fi, err := os.Stat(preparePath(statDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errVolumeNotFound
		} else if os.IsPermission(err) {
			return nil, errVolumeAccessDenied
		}
		return nil, err
	}

	if !fi.IsDir() {
		return nil, errVolumeAccessDenied
	}

	return fi, nil
}

// Lookup if file exists, returns file attributes upon success
func fsStatFile(statFile string) (os.FileInfo, error) {
	if statFile == "" {
		return nil, errInvalidArgument
	}

	if err := checkPathLength(statFile); err != nil {
		return nil, err
	}

	fi, err := os.Stat(preparePath(statFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errFileNotFound
		} else if os.IsPermission(err) {
			return nil, errFileAccessDenied
		} else if isSysErrNotDir(err) {
			return nil, errFileAccessDenied
		} else if isSysErrPathNotFound(err) {
			return nil, errFileNotFound
		}
		return nil, err
	}
	if fi.IsDir() {
		return nil, errFileNotFound
	}
	return fi, nil
}

// Opens the file at given path, optionally from an offset. Upon success returns
// a readable stream and the size of the readable stream.
func fsOpenFile(readPath string, offset int64) (io.ReadCloser, int64, error) {
	if readPath == "" || offset < 0 {
		return nil, 0, errInvalidArgument
	}
	if err := checkPathLength(readPath); err != nil {
		return nil, 0, err
	}

	fr, err := os.Open(preparePath(readPath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, errFileNotFound
		} else if os.IsPermission(err) {
			return nil, 0, errFileAccessDenied
		} else if isSysErrNotDir(err) {
			// File path cannot be verified since one of the parents is a file.
			return nil, 0, errFileAccessDenied
		} else if isSysErrPathNotFound(err) {
			// Add specific case for windows.
			return nil, 0, errFileNotFound
		}
		return nil, 0, err
	}

	// Stat to get the size of the file at path.
	st, err := fr.Stat()
	if err != nil {
		return nil, 0, err
	}

	// Verify if its not a regular file, since subsequent Seek is undefined.
	if !st.Mode().IsRegular() {
		return nil, 0, errIsNotRegular
	}

	// Seek to the requested offset.
	if offset > 0 {
		_, err = fr.Seek(offset, os.SEEK_SET)
		if err != nil {
			return nil, 0, err
		}
	}

	// Success.
	return fr, st.Size(), nil
}

// Creates a file and copies data from incoming reader. Staging buffer is used by io.CopyBuffer.
func fsCreateFile(tempObjPath string, reader io.Reader, buf []byte, fallocSize int64) (int64, error) {
	if tempObjPath == "" || reader == nil || buf == nil {
		return 0, errInvalidArgument
	}

	if err := checkPathLength(tempObjPath); err != nil {
		return 0, err
	}

	if err := mkdirAll(pathutil.Dir(tempObjPath), 0777); err != nil {
		return 0, err
	}

	writer, err := os.OpenFile(preparePath(tempObjPath), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		// File path cannot be verified since one of the parents is a file.
		if isSysErrNotDir(err) {
			return 0, errFileAccessDenied
		}
		return 0, err
	}
	defer writer.Close()

	// Fallocate only if the size is final object is known.
	if fallocSize > 0 {
		if err = fsFAllocate(int(writer.Fd()), 0, fallocSize); err != nil {
			return 0, err
		}
	}

	bytesWritten, err := io.CopyBuffer(writer, reader, buf)
	if err != nil {
		return 0, err
	}

	return bytesWritten, nil
}

// Removes uploadID at destination path.
func fsRemoveUploadIDPath(basePath, uploadIDPath string) error {
	if basePath == "" || uploadIDPath == "" {
		return errInvalidArgument
	}

	// List all the entries in uploadID.
	entries, err := readDir(uploadIDPath)
	if err != nil && err != errFileNotFound {
		return err
	}

	// Delete all the entries obtained from previous readdir.
	for _, entryPath := range entries {
		err = fsDeleteFile(basePath, pathJoin(uploadIDPath, entryPath))
		if err != nil && err != errFileNotFound {
			return err
		}
	}

	return nil
}

// fsFAllocate is similar to Fallocate but provides a convenient
// wrapper to handle various operating system specific errors.
func fsFAllocate(fd int, offset int64, len int64) (err error) {
	e := Fallocate(fd, offset, len)
	// Ignore errors when Fallocate is not supported in the current system
	if e != nil && !isSysErrNoSys(e) && !isSysErrOpNotSupported(e) {
		switch {
		case isSysErrNoSpace(e):
			err = errDiskFull
		case isSysErrIO(e):
			err = e
		default:
			// For errors: EBADF, EINTR, EINVAL, ENODEV, EPERM, ESPIPE  and ETXTBSY
			// Appending was failed anyway, returns unexpected error
			err = errUnexpected
		}
		return err
	}

	return nil
}

// Renames source path to destination path, creates all the
// missing parents if they don't exist.
func fsRenameFile(sourcePath, destPath string) error {
	if err := mkdirAll(pathutil.Dir(destPath), 0777); err != nil {
		return traceError(err)
	}
	if err := os.Rename(preparePath(sourcePath), preparePath(destPath)); err != nil {
		return traceError(err)
	}
	return nil
}

// Delete a file and its parent if it is empty at the destination path.
// this function additionally protects the basePath from being deleted.
func fsDeleteFile(basePath, deletePath string) error {
	if err := checkPathLength(basePath); err != nil {
		return err
	}

	if err := checkPathLength(deletePath); err != nil {
		return err
	}

	if basePath == deletePath {
		return nil
	}

	// Verify if the path exists.
	pathSt, err := os.Stat(preparePath(deletePath))
	if err != nil {
		if os.IsNotExist(err) {
			return errFileNotFound
		} else if os.IsPermission(err) {
			return errFileAccessDenied
		}
		return err
	}

	if pathSt.IsDir() && !isDirEmpty(deletePath) {
		// Verify if directory is empty.
		return nil
	}

	// Attempt to remove path.
	if err = os.Remove(preparePath(deletePath)); err != nil {
		if os.IsNotExist(err) {
			return errFileNotFound
		} else if os.IsPermission(err) {
			return errFileAccessDenied
		} else if isSysErrNotEmpty(err) {
			return errVolumeNotEmpty
		}
		return err
	}

	// Recursively go down the next path and delete again.
	if err := fsDeleteFile(basePath, pathutil.Dir(deletePath)); err != nil {
		return err
	}

	return nil
}
