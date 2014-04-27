package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	defaultPrefix = "_ver"
)

func Usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [flags] ver [rev]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	var prefix string
	var msg string
	var noCommit bool
	var force bool

	flag.StringVar(&prefix, "prefix", defaultPrefix, "the path prefix for vers")
	flag.StringVar(&msg, "m", "", "used instead of the default commit message")
	flag.BoolVar(&noCommit, "n", false, "don't commit the updated ver tree")
	flag.BoolVar(&force, "f", false, "force an existing ver to be updated")
	flag.Usage = Usage
	flag.Parse()

	ver := flag.Arg(0)
	if ver == "" {
		Usage()
	}

	rev := flag.Arg(1)
	if rev == "" {
		rev = ver
	}

	// Verify rev
	revSha := GitRun("rev-parse", "--verify", fmt.Sprintf("%s^{commit}", rev))

	verpath := filepath.Join(prefix, ver)

	// Make sure the index is clear
	if GitIndexDirty() {
		fmt.Fprint(os.Stderr, "Index already has staged changes. Cannot continue.")
		os.Exit(3)
	}

	// Remove existing verpath if force is set, otherwise fail on existing path
	if _, err := os.Stat(verpath); !os.IsNotExist(err) {
		if force {
			GitRm(verpath)
		} else {
			fmt.Fprintf(os.Stderr, "%s already exists; override with -f\n", verpath)
			os.Exit(4)
		}
	}

	GitRun("read-tree", "-u", fmt.Sprintf("--prefix=%s", verpath), rev)

	GitRm(filepath.Join(verpath, prefix))

	if !GitIndexDirty() {
		fmt.Fprint(os.Stderr, "Nothing to commit\n")
	} else if !noCommit {
		commitArgs := []string{"commit", "--no-verify"}
		if msg == "" {
			msg = fmt.Sprintf("Update ver %s to %s", ver, revSha)
			commitArgs = append(commitArgs, "--edit")
		}
		commitArgs = append(commitArgs, "-m", msg)
		commitCmd := exec.Command("git", commitArgs...)
		commitCmd.Stdin = os.Stdin
		commitCmd.Stdout = os.Stdout
		commitCmd.Stderr = os.Stderr
		err := commitCmd.Run()
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(5)
		}
	}
}

func GitCmd(args ...string) ([]byte, error) {
	return exec.Command("git", args...).CombinedOutput()
}

func GitRun(args ...string) string {
	out, err := GitCmd(args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "`git %s` failed:\n", strings.Join(args, " "))
		os.Stderr.Write(out)
		os.Exit(127)
	}
	return string(bytes.TrimSpace(out))
}

func GitIndexDirty() bool {
	_, err := GitCmd("diff-index", "--cached", "--exit-code", "HEAD")
	switch err.(type) {
	case nil:
		return false
	case *exec.ExitError:
		return true
	default:
		fmt.Fprint(os.Stderr, err)
		os.Exit(126)
		return false
	}
}

func GitRm(path string) {
	GitRun("rm", "-r", "--force", "--ignore-unmatch", "--", path)
}
