// Copyright 2021 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

// WriteTextFile overwrites the file on the given path with content
func WriteTextFile(filePath, content string) error {
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content += "\n"
	}
	return ioutil.WriteFile(filePath, []byte(content), 0600)
}

// GitRootDir returns the absolute path to the root directory of the git repo
// where this function is called
func GitRootDir() (string, error) {
	dir, err := Shell("git rev-parse --show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.Trim(dir, "\n"), nil
}

// Poll executes do() after time interval for a max of numTrials times.
// The bool returned by do() indicates if polling succeeds in that trial
func Poll(interval time.Duration, numTrials int, do func() (bool, error)) error {
	if numTrials < 0 {
		return fmt.Errorf("numTrials cannot be negative")
	}
	for i := 0; i < numTrials; i++ {
		if success, err := do(); err != nil {
			return fmt.Errorf("error during trial %d: %v", i, err)
		} else if success {
			return nil
		} else {
			time.Sleep(interval)
		}
	}
	return fmt.Errorf("max polling iteration reached")
}

// CreateTempfile creates a tempfile string.
func CreateTempfile(tmpDir, prefix, suffix string) (string, error) {
	f, err := ioutil.TempFile(tmpDir, prefix)
	if err != nil {
		return "", err
	}
	var tmpName string
	if tmpName, err = filepath.Abs(f.Name()); err != nil {
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	if err = os.Remove(tmpName); err != nil {
		log.Log.Errorf("CreateTempfile unable to remove %s", tmpName)
		return "", err
	}
	return tmpName + suffix, nil
}

// WriteTempfile creates a tempfile with the specified contents.
func WriteTempfile(tmpDir, prefix, suffix, contents string) (string, error) {
	fname, err := CreateTempfile(tmpDir, prefix, suffix)
	if err != nil {
		return "", err
	}

	if err := ioutil.WriteFile(fname, []byte(contents), 0644); err != nil {
		return "", err
	}
	return fname, nil
}

// Shell run command on shell and get back output and error if get one
func Shell(format string, args ...interface{}) (string, error) {
	return sh(context.Background(), format, true, true, true, "", args...)
}

// Shell runs command on shell, passing in the specified reader as the stdin, and get back output and error if get one
func ShellWithInput(input string, format string, args ...interface{}) (string, error) {
	return sh(context.Background(), format, true, true, true, input, args...)
}

// ShellContext run command on shell and get back output and error if get one
func ShellCtx(ctx context.Context, format string, args ...interface{}) (string, error) {
	return sh(ctx, format, true, true, true, "", args...)
}

// ShellMuteOutput run command on shell and get back output and error if get one
// without logging the output
func ShellMuteOutput(format string, args ...interface{}) (string, error) {
	return sh(context.Background(), format, true, false, true, "", args...)
}

// ShellMuteOutputError run command on shell and get back output and error if get one
// without logging the output or errors
func ShellMuteOutputError(format string, args ...interface{}) (string, error) {
	return sh(context.Background(), format, true, false, false, "", args...)
}

// ShellSilent runs command on shell and get back output and error if get one
// without logging the command or output.
func ShellSilent(format string, args ...interface{}) (string, error) {
	return sh(context.Background(), format, false, false, false, "", args...)
}

func sh(ctx context.Context, format string, logCommand, logOutput, logError bool, input string, args ...interface{}) (string, error) {
	command := fmt.Sprintf(format, args...)
	if logCommand {
		log.Log.Infof("Running command: %s", command)
	}
	c := exec.CommandContext(ctx, "sh", "-c", command) // #nosec
	if input != "" {
		log.Log.Infof("Command input:\n%s", input)
		c.Stdin = strings.NewReader(input)
	}
	bytes, err := c.CombinedOutput()
	if logOutput {
		if output := strings.TrimSuffix(string(bytes), "\n"); len(output) > 0 {
			log.Log.Infof("Command output: \n%s", output)
		}
	}

	if err != nil {
		if logError {
			log.Log.Infof("Command error: %v", err)
		}
		return string(bytes), fmt.Errorf("command failed: %s", string(bytes))
	}
	return string(bytes), nil
}

// RunBackground starts a background process and return the Process if succeed
func RunBackground(format string, args ...interface{}) (*os.Process, error) {
	command := fmt.Sprintf(format, args...)
	log.Log.Info("RunBackground: ", command)
	parts := strings.Split(command, " ")
	c := exec.Command(parts[0], parts[1:]...) // #nosec
	err := c.Start()
	if err != nil {
		log.Log.Errorf("%s, command failed!", command)
		return nil, err
	}
	return c.Process, nil
}

// Record run command and record output into a file
func Record(command, record string) error {
	resp, err := Shell(command)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(record, []byte(resp), 0600)
	return err
}

// HTTPDownload download from src(url) and store into dst(local file)
func HTTPDownload(dst string, src string) error {
	log.Log.Infof("Start downloading from %s to %s ...\n", src, dst)
	var err error
	var out *os.File
	var resp *http.Response
	out, err = os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err = out.Close(); err != nil {
			log.Log.Errorf("Error: close file %s, %s", dst, err)
		}
	}()
	resp, err = http.Get(src)
	if err != nil {
		return err
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Log.Errorf("Error: close downloaded file from %s, %s", src, err)
		}
	}()
	if resp.StatusCode != 200 {
		return fmt.Errorf("http get request, received unexpected response status: %s", resp.Status)
	}
	if _, err = io.Copy(out, resp.Body); err != nil {
		return err
	}
	log.Log.Info("Download successfully!")
	return err
}

// GetOsExt returns the current OS tag.
func GetOsExt() (string, error) {
	var osExt string
	switch runtime.GOOS {
	case "linux":
		osExt = "linux"
	case "darwin":
		osExt = "osx"
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return osExt, nil
}

// CopyFile create a new file to src based on dst
func CopyFile(src, dst string) error {
	var in, out *os.File
	var err error
	in, err = os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err = in.Close(); err != nil {
			log.Log.Errorf("Error: close file from %s, %s", src, err)
		}
	}()
	out, err = os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err = out.Close(); err != nil {
			log.Log.Errorf("Error: close file from %s, %s", dst, err)
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	err = out.Sync()
	return err
}

// ExtractTarGz extracts a .tar.gz file into current dir.
func ExtractTarGz(gzipStream io.Reader) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return errors.Wrap(err, "Fail to uncompress")
	}
	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return errors.Wrap(err, "ExtractTarGz: Next() failed")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(header.Name, 0755); err != nil {
				return errors.Wrap(err, "ExtractTarGz: Mkdir() failed")
			}
		case tar.TypeReg:
			outFile, err := os.Create(header.Name)
			if err != nil {
				return errors.Wrap(err, "ExtractTarGz: Create() failed")
			}
			defer outFile.Close() // nolint: errcheck
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrap(err, "ExtractTarGz: Copy() failed")
			}
		default:
			return fmt.Errorf("unknown type: %s in %s",
				string(header.Typeflag), header.Name)
		}
	}
	return nil
}
