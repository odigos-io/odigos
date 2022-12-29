package goobj2

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func getNewObjPath(objPath string) string {
	return filepath.Join(filepath.Dir(objPath), "new_"+filepath.Base(objPath))
}

type test struct {
	name string
	path string
	pkg  string
	obj  bool
}

func TestWrite(t *testing.T) {
	var tests []test

	filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Fatalf("failed to walk testdata dir: %v", err)
		}

		if info.IsDir() {
			return nil
		}

		tests = append(tests, test{info.Name(), path, "", false})

		return nil
	})

	tempDir := t.TempDir()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basename := strings.TrimSuffix(tt.name, filepath.Ext(tt.name))
			var objPath string
			if tt.obj {
				objPath = tt.path
			} else {
				objPath = filepath.Join(tempDir, basename+".o")
				cmd := exec.Command("go", "tool", "compile", "-o", objPath, tt.path)
				if err := cmd.Run(); err != nil {
					t.Fatalf("failed to compile: %v", err)
				}
			}

			// parse obj file
			pkg, err := Parse(objPath, tt.pkg)
			if err != nil {
				t.Fatalf("failed to parse object file: %v", err)
			}
			//ioutil.WriteFile(objPath+"_parsed", []byte(pretty.Sprint(pkg)), 0777)

			// write obj file
			newObjPath := getNewObjPath(objPath)
			WriteObjFile2(pkg, newObjPath)

			// compare bytes of the original and written object files
			objBytes, err := ioutil.ReadFile(objPath)
			if err != nil {
				t.Fatalf("failed to read object file: %v", err)
			}
			newObjBytes, err := ioutil.ReadFile(newObjPath)
			if err != nil {
				t.Fatalf("failed to read new object file: %v", err)
			}

			if !bytes.Equal(objBytes, newObjBytes) {
				t.Error("object files are not the same")
			}

			// compare parsed packages of the two object files
			_, err = Parse(newObjPath, tt.pkg)
			if err != nil {
				t.Fatalf("failed to parse new object file: %v", err)
			}

			/*if !reflect.DeepEqual(pkg, pkg2) {
				t.Errorf("Packages are not equal:\n%v", strings.Join(pretty.Diff(pkg, pkg2), "\n"))
			}*/
		})
	}
}
