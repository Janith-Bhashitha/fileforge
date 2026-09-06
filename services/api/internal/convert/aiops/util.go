package aiops

import (
	"errors"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
)

func outputPath(dir, prefix, ext string) string {
	return filepath.Join(dir, prefix+"-"+uuid.New().String()+ext)
}

// asExitError is errors.As with the signature the callers here want -
// subprocess failures carry their diagnostics on Stderr, and losing that
// turns "tesseract couldn't find the language pack" into "exit status 1".
func asExitError(err error, target **exec.ExitError) bool {
	return errors.As(err, target)
}
