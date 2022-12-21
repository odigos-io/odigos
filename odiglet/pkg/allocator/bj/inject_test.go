package bj

import (
	"bytes"
	"io"
	"log"
	"os"
	"testing"
)

func CompareFiles(file1, file2 string) bool {
	const chunkSize = 64 * 1024
	f1, err := os.Open(file1)
	if err != nil {
		log.Fatal(err)
	}
	f2, err := os.Open(file2)
	if err != nil {
		log.Fatal(err)
	}

	for {
		b1 := make([]byte, chunkSize)
		_, err1 := f1.Read(b1)

		b2 := make([]byte, chunkSize)
		_, err2 := f2.Read(b2)

		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true
			} else if err1 == io.EOF || err2 == io.EOF {
				return false
			} else {
				log.Fatal(err1, err2)
			}
		}

		if !bytes.Equal(b1, b2) {
			return false
		}
	}
}

func Test_Elf_Inject_Static_Nop_1(t *testing.T) {

	os.Mkdir("tmp", 0755)
	err := BinjectFile("test/static_ls", "tmp/static_ls_injected", "test/nop.bin", &BinjectConfig{CodeCaveMode: false, InjectionMethod: SilvioInject})
	if err != nil {
		t.Error(err)
	}

	if !CompareFiles("test/static_ls_nop_injected", "tmp/static_ls_injected") {
		t.Error("Generated File Did Not Match!")
	} else {
		t.Log("Shellcode Injected Successfully!")
	}
	os.RemoveAll("tmp")
}

func Test_Elf_Inject_Exec_Hello_1(t *testing.T) {

	os.Mkdir("tmp", 0755)
	err := BinjectFile("test/static_ls", "tmp/static_ls_injected", "test/hello.bin", &BinjectConfig{CodeCaveMode: false, InjectionMethod: SilvioInject})
	if err != nil {
		t.Error(err)
	}

	if !CompareFiles("test/static_ls_hello_injected", "tmp/static_ls_injected") {
		t.Error("Generated File Did Not Match!")
	} else {
		t.Log("Shellcode Injected Successfully!")
	}
	os.RemoveAll("tmp")
}

func Test_Elf_Inject_Exec_PTNOTE_Hello_1(t *testing.T) {

	os.Mkdir("tmp", 0755)
	err := BinjectFile("test/static_ls", "tmp/ls_ptnote_hallo.injected", "test/hallo.bin", &BinjectConfig{CodeCaveMode: false, InjectionMethod: PtNoteInject})
	if err != nil {
		t.Error(err)
	}

	if !CompareFiles("test/ls_ptnote_hallo", "tmp/ls_ptnote_hallo.injected") {
		t.Error("Generated File Did Not Match!")
	} else {
		t.Log("Shellcode Injected Successfully!")
	}
	os.RemoveAll("tmp")
}
