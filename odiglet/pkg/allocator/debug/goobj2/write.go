// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Writing Go object files.

// This file is a modified version of cmd/internal/obj/objfile2.go

package goobj2

import (
	"bytes"
	"fmt"
	"github.com/keyval-dev/odigos/odiglet/pkg/allocator/debug/goobj2/internal/bio"
	"github.com/keyval-dev/odigos/odiglet/pkg/allocator/debug/goobj2/internal/goobj2"
	"path/filepath"
	"strings"
)

// Write writes the contents of the parsed archive to disk.
func (pkg *Package) Write(path string) (err error) {
	b, err := bio.Create(path)
	if err != nil {
		return fmt.Errorf("error creating object file: %v", err)
	}
	defer func() {
		closeErr := b.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	// Archive headers
	b.Write(archiveHeader)
	var arhdr [archiveHeaderLen]byte
	var curArHdrOff, curObjStartOff int64
	for i := range pkg.ArchiveMembers {
		ctxt := &pkg.ArchiveMembers[i]
		ar := ctxt.ArchiveHeader
		curArHdrOff = b.Offset()

		copy(arhdr[:], fmt.Sprintf("%-16s%-12s%-6s%-6s%-8s%-10d`\n", ar.Name, ar.Date, ar.UID, ar.GID, ar.Mode, ar.Size))
		b.Write(arhdr[:])
		curObjStartOff = b.Offset()
		b.Write(ar.Data)
		if ctxt.IsDataObj {
			continue
		}

		genFuncInfoSyms(ctxt)

		w := writer{
			Writer: goobj2.NewWriter(b),
			ctxt:   ctxt,
		}

		start := b.Offset()

		// Header
		// We just reserve the space. We'll fill in the offsets later.
		ctxt.ObjHeader.Write(w.Writer)

		// String table
		w.StringTable()

		// Autolib
		ctxt.ObjHeader.Offsets[goobj2.BlkAutolib] = w.Offset()
		for i := range ctxt.Imports {
			ctxt.Imports[i].Write(w.Writer)
		}

		// Package references
		ctxt.ObjHeader.Offsets[goobj2.BlkPkgIdx] = w.Offset()
		w.StringRef("")
		for _, pkg := range ctxt.Packages {
			w.StringRef(pkg)
		}

		// DWARF file table
		ctxt.ObjHeader.Offsets[goobj2.BlkDwarfFile] = w.Offset()
		for _, f := range ctxt.DWARFFileList {
			w.StringRef(filepath.ToSlash(f))
		}

		// Symbol definitions
		ctxt.ObjHeader.Offsets[goobj2.BlkSymdef] = w.Offset()
		for _, s := range ctxt.SymDefs {
			w.Sym(s)
		}

		// Non-pkg symbol definitions
		ctxt.ObjHeader.Offsets[goobj2.BlkNonpkgdef] = w.Offset()
		for _, s := range ctxt.NonPkgSymDefs {
			w.Sym(s)
		}

		// Non-pkg symbol references
		ctxt.ObjHeader.Offsets[goobj2.BlkNonpkgref] = w.Offset()
		for _, s := range ctxt.NonPkgSymRefs {
			w.Sym(s)
		}

		// Reloc indexes
		ctxt.ObjHeader.Offsets[goobj2.BlkRelocIdx] = w.Offset()
		nreloc := uint32(0)
		lists := [][]*Sym{ctxt.SymDefs, ctxt.NonPkgSymDefs}
		for _, list := range lists {
			for _, s := range list {
				w.Uint32(nreloc)
				nreloc += uint32(len(s.Reloc))
			}
		}
		w.Uint32(nreloc)

		// Symbol Info indexes
		ctxt.ObjHeader.Offsets[goobj2.BlkAuxIdx] = w.Offset()
		naux := uint32(0)
		for _, list := range lists {
			for _, s := range list {
				w.Uint32(naux)
				naux += uint32(nAuxSym(s))
			}
		}
		w.Uint32(naux)

		// Data indexes
		ctxt.ObjHeader.Offsets[goobj2.BlkDataIdx] = w.Offset()
		dataOff := uint32(0)
		for _, list := range lists {
			for _, s := range list {
				w.Uint32(dataOff)
				dataOff += uint32(len(s.Data))
			}
		}
		w.Uint32(dataOff)

		// Relocs
		ctxt.ObjHeader.Offsets[goobj2.BlkReloc] = w.Offset()
		for _, list := range lists {
			for _, s := range list {
				for i := range s.Reloc {
					w.Reloc(&s.Reloc[i])
				}
			}
		}

		// Aux symbol info
		ctxt.ObjHeader.Offsets[goobj2.BlkAux] = w.Offset()
		for _, list := range lists {
			for _, s := range list {
				w.Aux(s)
			}
		}

		// Data
		ctxt.ObjHeader.Offsets[goobj2.BlkData] = w.Offset()
		for _, list := range lists {
			for _, s := range list {
				w.Bytes(s.Data)
			}
		}

		// Pcdata
		ctxt.ObjHeader.Offsets[goobj2.BlkPcdata] = w.Offset()
		for _, ts := range ctxt.textSyms {
			w.Bytes(ts.Func.PCSP)
			w.Bytes(ts.Func.PCFile)
			w.Bytes(ts.Func.PCLine)
			w.Bytes(ts.Func.PCInline)
			for i := range ts.Func.PCData {
				w.Bytes(ts.Func.PCData[i])
			}
		}

		// Referenced symbol names from other packages
		ctxt.ObjHeader.Offsets[goobj2.BlkRefName] = w.Offset()
		for _, ref := range ctxt.SymRefs {
			var o goobj2.RefName
			o.SetSym(ref.SymRef)
			o.SetName(ref.Name, w.Writer)
			o.Write(w.Writer)
		}

		objEnd := w.Offset()
		ctxt.ObjHeader.Offsets[goobj2.BlkEnd] = objEnd

		// If the object size is odd, make it even by adding an
		// extra null byte as padding
		size := int64(objEnd) + (start - curObjStartOff)
		end := start + int64(w.Offset())
		if size%2 != 0 {
			b.WriteByte(0x00)
			end++
		}

		// Fix size field of the last archive header
		b.MustSeek(curArHdrOff+48, 0)
		b.WriteString(fmt.Sprintf("%-10d", size))

		// Fix up block offsets in the object header
		b.MustSeek(start, 0)
		ctxt.ObjHeader.Write(w.Writer)
		b.MustSeek(end, 0)
	}

	return nil
}

type writer struct {
	*goobj2.Writer
	ctxt *ArchiveMember
}

func (w *writer) StringTable() {
	w.AddString("")
	for _, p := range w.ctxt.Imports {
		w.AddString(p.Pkg)
	}
	for _, pkg := range w.ctxt.Packages {
		w.AddString(pkg)
	}

	writeSymStrings := func(s *Sym) {
		w.AddString(s.Name)

		for _, r := range s.Reloc {
			w.AddString(r.Name)
		}
		if s.Type != nil {
			w.AddString(s.Name)
		}

		if s.Kind == STEXT && s.Func != nil {
			for _, d := range s.Func.FuncData {
				w.AddString(d.Sym.Name)
			}
			for _, f := range s.Func.File {
				w.AddString(filepath.ToSlash(f.Name))
			}
			for _, call := range s.Func.InlTree {
				w.AddString(call.File.Name)
				w.AddString(call.Func.Name)
			}

			dwsyms := []*SymRef{s.Func.DwarfRanges, s.Func.DwarfLoc, s.Func.DwarfDebugLines, s.Func.FuncInfo}
			for _, dws := range dwsyms {
				if dws != nil {
					w.AddString(dws.Name)
				}
			}
		}
	}

	// Symbols of type STEXT (that have functions) are written first
	for _, ts := range w.ctxt.textSyms {
		writeSymStrings(ts)
	}

	syms := [][]*Sym{w.ctxt.NonPkgSymDefs, w.ctxt.SymDefs, w.ctxt.NonPkgSymRefs}
	for _, list := range syms {
		for _, s := range list {
			if s.Kind == STEXT {
				continue
			}

			writeSymStrings(s)
		}
	}
	for _, r := range w.ctxt.SymRefs {
		w.AddString(r.Name)
	}

	for _, f := range w.ctxt.DWARFFileList {
		w.AddString(filepath.ToSlash(f))
	}
}

func (w *writer) Sym(s *Sym) {
	name := s.Name
	if strings.HasPrefix(name, "gofile..") {
		name = filepath.ToSlash(name)
	}

	var o goobj2.Sym
	o.SetName(name, w.Writer)
	o.SetABI(s.ABI)
	o.SetType(uint8(s.Kind))
	o.SetFlag(s.Flag)
	o.SetSiz(s.Size)
	o.SetAlign(s.Align)
	o.Write(w.Writer)
}

func (w *writer) Reloc(r *Reloc) {
	var o goobj2.Reloc
	o.SetOff(int32(r.Offset))
	o.SetSiz(uint8(r.Size))
	o.SetType(uint8(r.Type))
	o.SetAdd(r.Add)
	o.SetSym(r.Sym)
	o.Write(w.Writer)
}

func (w *writer) aux1(typ uint8, rs goobj2.SymRef) {
	var o goobj2.Aux
	o.SetType(typ)
	o.SetSym(rs)
	o.Write(w.Writer)
}

func (w *writer) Aux(s *Sym) {
	if s.Type != nil {
		w.aux1(goobj2.AuxGotype, s.Type.SymRef)
	}
	if s.Func != nil {
		w.aux1(goobj2.AuxFuncInfo, s.Func.FuncInfo.SymRef)

		for _, d := range s.Func.FuncData {
			w.aux1(goobj2.AuxFuncdata, d.Sym.SymRef)
		}

		if s.Func.DwarfInfo != nil {
			w.aux1(goobj2.AuxDwarfInfo, s.Func.DwarfInfo.SymRef)
		}
		if s.Func.DwarfLoc != nil {
			w.aux1(goobj2.AuxDwarfLoc, s.Func.DwarfLoc.SymRef)
		}
		if s.Func.DwarfRanges != nil {
			w.aux1(goobj2.AuxDwarfRanges, s.Func.DwarfRanges.SymRef)
		}
		if s.Func.DwarfDebugLines != nil {
			w.aux1(goobj2.AuxDwarfLines, s.Func.DwarfDebugLines.SymRef)
		}
	}
}

// return the number of aux symbols s have.
func nAuxSym(s *Sym) int {
	n := 0
	if s.Type != nil {
		n++
	}
	if s.Func != nil {
		// FuncInfo is an aux symbol, each Funcdata is an aux symbol
		n += 1 + len(s.Func.FuncData)
		if s.Func.DwarfInfo != nil {
			n++
		}
		if s.Func.DwarfLoc != nil {
			n++
		}
		if s.Func.DwarfRanges != nil {
			n++
		}
		if s.Func.DwarfDebugLines != nil {
			n++
		}
	}
	return n
}

// generate symbols for FuncInfo.
func genFuncInfoSyms(ctxt *ArchiveMember) {
	var pcdataoff uint32
	var b bytes.Buffer
	for _, s := range ctxt.textSyms {
		if s.Func == nil {
			continue
		}

		o := goobj2.FuncInfo{
			Args:   uint32(s.Func.Args),
			Locals: uint32(s.Func.Frame),
		}
		o.Pcsp = pcdataoff
		pcdataoff += uint32(len(s.Func.PCSP))
		o.Pcfile = pcdataoff
		pcdataoff += uint32(len(s.Func.PCFile))
		o.Pcline = pcdataoff
		pcdataoff += uint32(len(s.Func.PCLine))
		o.Pcinline = pcdataoff
		pcdataoff += uint32(len(s.Func.PCInline))
		o.Pcdata = make([]uint32, len(s.Func.PCData))
		for i, pcd := range s.Func.PCData {
			o.Pcdata[i] = pcdataoff
			pcdataoff += uint32(len(pcd))
		}
		o.PcdataEnd = pcdataoff
		o.Funcdataoff = make([]uint32, len(s.Func.FuncData))
		for i, x := range s.Func.FuncData {
			o.Funcdataoff[i] = x.Offset
		}
		o.File = make([]goobj2.SymRef, len(s.Func.File))
		for i, f := range s.Func.File {
			o.File[i] = f.SymRef
		}
		o.InlTree = make([]goobj2.InlTreeNode, len(s.Func.InlTree))
		for i, inl := range s.Func.InlTree {
			o.InlTree[i] = goobj2.InlTreeNode{
				Parent:   inl.Parent,
				File:     inl.File.SymRef,
				Line:     inl.Line,
				Func:     inl.Func.SymRef,
				ParentPC: inl.ParentPC,
			}
		}

		o.Write(&b)
		ctxt.symMap[s.Func.dataSymIdx].Data = append([]byte(nil), b.Bytes()...)
		b.Reset()
	}
}
