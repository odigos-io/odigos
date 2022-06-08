package process

import (
	"fmt"
	"github.com/fntlnz/mountinfo"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

type Details struct {
	ProcessID int
	ExeName   string
	CmdLine   string
}

func FindAllInContainer(podUID string, containerName string) ([]*Details, error) {
	proc, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}

	var result []*Details
	for {
		dirs, err := proc.Readdir(15)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		for _, di := range dirs {
			if !di.IsDir() {
				continue
			}

			dname := di.Name()
			if dname[0] < '0' || dname[0] > '9' {
				continue
			}

			pid, err := strconv.Atoi(dname)
			if err != nil {
				return nil, err
			}

			mi, err := mountinfo.GetMountInfo(path.Join("/proc", dname, "mountinfo"))
			if err != nil {
				continue
			}

			for _, m := range mi {
				root := m.Root
				if strings.Contains(root, fmt.Sprintf("%s/containers/%s", podUID, containerName)) {
					exeName, err := os.Readlink(path.Join("/proc", dname, "exe"))
					if err != nil {
						// Read link may fail if target process runs not as root
						exeName = ""
					}

					cmdLine, err := ioutil.ReadFile(path.Join("/proc", dname, "cmdline"))
					var cmd string
					if err != nil {
						// Ignore errors
						cmd = ""
					} else {
						cmd = string(cmdLine)
					}

					result = append(result, &Details{
						ProcessID: pid,
						ExeName:   exeName,
						CmdLine:   cmd,
					})
				}
			}
		}
	}

	return result, nil
}
