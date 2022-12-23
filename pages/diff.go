package pages

import (
	"io"
	"os"
	"os/exec"
	"strings"
)

const posixDiff = "diff"

func (p1 Page) Diff(p2 *Page) (string, error) {
	t1, err := os.CreateTemp("", "wikipage")
	if err != nil {
		return "", err
	}
	defer os.Remove(t1.Name())
	t2, err := os.CreateTemp("", "wikipage")
	if err != nil {
		return "", err
	}
	defer os.Remove(t2.Name())
	var buf strings.Builder

	// Just invoke the system diff command, we can't return a HashDiff
	// since we're not working things that are tracked by the repo.
	// we just directly invoke diff if --no-index is specified.
	if p2 == nil {
		diffcmd := exec.Command(posixDiff, "-U", "5", "-L", "/dev/null", "-L", "New Title", "/dev/null", t1.Name())
		diffcmd.Stderr = os.Stderr
		diffcmd.Stdout = &buf
		t1.Truncate(0)
		t1.Seek(0, 0)
		if _, err := io.WriteString(t1, p1.Title+"\n"); err != nil {
			return "", err
		}
		diffcmd.Run()
		io.WriteString(&buf, "\n")

		diffcmd = exec.Command(posixDiff, "-U", "5", "-L", "/dev/null", "-L", "New Summary", "/dev/null", t1.Name())
		diffcmd.Stderr = os.Stderr
		diffcmd.Stdout = &buf
		t1.Truncate(0)
		t1.Seek(0, 0)
		if _, err := io.WriteString(t1, p1.Summary+"\n"); err != nil {
			return "", err
		}
		diffcmd.Run()
		io.WriteString(&buf, "\n")

		diffcmd = exec.Command(posixDiff, "-U", "5", "-L", "/dev/null", "-L", "New Content", "/dev/null", t1.Name())
		diffcmd.Stderr = os.Stderr
		diffcmd.Stdout = &buf
		t1.Truncate(0)
		t1.Seek(0, 0)
		if _, err := io.WriteString(t1, p1.Content+"\n"); err != nil {
			return "", err
		}
		diffcmd.Run()
		io.WriteString(&buf, "\n")
	} else {
		if p1.Title != p2.Title {
			diffcmd := exec.Command(posixDiff, "-U", "5", "-L", "Old Title", "-L", "New Title", t2.Name(), t1.Name())
			diffcmd.Stderr = os.Stderr
			diffcmd.Stdout = &buf
			t1.Truncate(0)
			t2.Truncate(0)
			t1.Seek(0, 0)
			t2.Seek(0, 0)
			if _, err := io.WriteString(t1, p1.Title+"\n"); err != nil {
				return "", err
			}
			if _, err := io.WriteString(t2, p2.Title+"\n"); err != nil {
				return "", err
			}
			diffcmd.Run()
			io.WriteString(&buf, "\n")
		}
		if p1.Summary != p2.Summary {
			diffcmd := exec.Command(posixDiff, "-U", "5", "-L", "Old Summary", "-L", "New Summary", t2.Name(), t1.Name())
			diffcmd.Stderr = os.Stderr
			diffcmd.Stdout = &buf
			t1.Truncate(0)
			t2.Truncate(0)
			t1.Seek(0, 0)
			t2.Seek(0, 0)
			if _, err := io.WriteString(t1, p1.Summary+"\n"); err != nil {
				return "", err
			}
			if _, err := io.WriteString(t2, p2.Summary+"\n"); err != nil {
				return "", err
			}
			diffcmd.Run()
			io.WriteString(&buf, "\n")
		}
		if p1.Content != p2.Content {
			diffcmd := exec.Command(posixDiff, "-U", "5", "-L", "Old Content", "-L", "New Content", t2.Name(), t1.Name())
			diffcmd.Stderr = os.Stderr
			diffcmd.Stdout = &buf
			t1.Truncate(0)
			t2.Truncate(0)
			t1.Seek(0, 0)
			t2.Seek(0, 0)
			if _, err := io.WriteString(t1, p1.Content+"\n"); err != nil {
				return "", err
			}
			if _, err := io.WriteString(t2, p2.Content+"\n"); err != nil {
				return "", err
			}
			diffcmd.Run()
			io.WriteString(&buf, "\n")
		}
	}
	return buf.String(), nil
}
