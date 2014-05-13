package TnT_final

import (
	"log"
	"os"
	"path/filepath"
)

func (tnt *TnTServer) PropagateUp(VersionVector map[int]int64, SyncVector map[int]int64, path string) {

	//path:path of the parent
	//Propagate the changes in a version vector upwards

	fst := tnt.Tree.MyTree // for ease of code
	//tnt.ParseTree("./",0)

	setMaxVersionVect(fst[path].VerVect, VersionVector)
	setMinVersionVect(fst[path].SyncVect, SyncVector)

	if path != "./" {
		tnt.PropagateUp(fst[path].VerVect, fst[path].SyncVect, fst[path].Parent)
	}
}

func (tnt *TnTServer) UpdateTreeWrapper(path string) {

	fst := tnt.Tree.MyTree

	tnt.Tree.LogicalTime += 1
	tnt.UpdateTree(path)

	if _, exists := fst[path]; exists == true {
		tnt.PropagateUp(fst[path].VerVect, fst[path].SyncVect, fst[path].Parent)
	} else {
		if path != "./" {
			prt := tnt.LiveAncestor(path)
			fst[prt].VerVect[tnt.me] = tnt.Tree.LogicalTime
			fst[prt].SyncVect[tnt.me] = tnt.Tree.LogicalTime
			tnt.PropagateUp(fst[prt].VerVect, fst[prt].SyncVect, fst[prt].Parent)
		}
	}
}

func (tnt *TnTServer) DeleteTree(path string) {
	// Deletes entire sub-tree under 'path' from FStree

	fst := tnt.Tree.MyTree

	if _, present := fst[path]; present {
		// Delete all children; recursively delete if child is a directory
		delete(fst[fst[path].Parent].Children, path)
		for child, _ := range fst[path].Children {
			tnt.DeleteTree(child)
		}
		delete(fst, path)
	}
}

func (tnt *TnTServer) UpdateTree(path string) {

	/* Remark: 'path' can be either a file or directory.

	(1) explores the file system, and make appropriate changes in FStree
	(2) deletes nodes in FStree which are not in the file system

	Caution: Should ensure that the first time UpdateTree is called, then
	'path' should exist in 'fst', and it's 'Parent' should be set appropriately.
	Otherwise, UpdateTree will create a spurious sub-tree which will not be
	reachable from the root. It is fine if stuff under 'path' are not in FST already.
	*/

	fst := tnt.Tree.MyTree

	fi, err := os.Lstat(tnt.root + path)

	if err != nil {
		tnt.DeleteTree(path)
		return
	}

	if _, present := fst[path]; present == false {
		fst[path] = new(FSnode)
		fst[path].Name = fi.Name()
		fst[path].Size = fi.Size()
		fst[path].IsDir = fi.IsDir()
		fst[path].LastModTime = fi.ModTime()
		fst[path].Creator = tnt.me
		fst[path].CreationTime = tnt.Tree.LogicalTime

		if fi.IsDir() {
			fst[path].Children = make(map[string]bool)
		}

		fst[path].VerVect = make(map[int]int64)
		fst[path].SyncVect = make(map[int]int64)
		for i:=0; i<len(tnt.servers); i++ {
			fst[path].VerVect[i] = 0
			fst[path].SyncVect[i] = 0
		}

		prt := parent(path)
		fst[path].Parent = prt
		fst[prt].Children[path] = true

		fst[path].VerVect[tnt.me] = tnt.Tree.LogicalTime
		// fst[path].SyncVect[tnt.me] = tnt.Tree.LogicalTime // set outside - unconditionally
	}

	fst[path].SyncVect[tnt.me] = tnt.Tree.LogicalTime

	if fi.IsDir() == false {
		if fst[path].LastModTime.Equal(fi.ModTime()) == false {
			fst[path].LastModTime = fi.ModTime()
			fst[path].VerVect[tnt.me] = tnt.Tree.LogicalTime
		}
	} else {
		/*
		if fst[path].LastModTime.Equal(fi.ModTime()) == false {
			fst[path].LastModTime = fi.ModTime()
			fst[path].VerVect[tnt.me] = tnt.Tree.LogicalTime
		}
		*/

		d, err := os.Open(tnt.root + path)
		defer d.Close()
		cfi, err := d.Readdir(-1)
		if err != nil {
			log.Println("LOL Error in UpdateTree:", err)
			os.Exit(1)
		}

		// Book-keeping : if fst.Tree[dir].Children[child] remains false in the end,
		// then it means child does not exist in file system
		for child, _ := range fst[path].Children {
			fst[path].Children[child] = false
		}

		for _, cfi := range cfi {
			var child string
			if cfi.IsDir() {
				child = path + cfi.Name() + string(filepath.Separator)
			} else {
				child = path + cfi.Name()
			}
			tnt.UpdateTree(child)
			fst[path].Children[child] = true
			//fst[child].Parent = path - the child does it :)

			// Update LastModTime, VerVect and SyncVect for 'dir' :

			// set my last mod time to be latest among all my children
			if fst[path].LastModTime.Before(fst[child].LastModTime) {
				fst[path].LastModTime = fst[child].LastModTime
			}

			setMaxVersionVect(fst[path].VerVect, fst[child].VerVect)  // VerVect is element-wise maximum of children's VerVect
			setMinVersionVect(fst[path].SyncVect, fst[child].SyncVect) // SyncVect is element-wise minimum of children's SyncVect
		}

		// Delete all my children, who were deleted since the last sync
		for child, exists := range fst[path].Children {
			if exists == false {
				tnt.DeleteTree(child)
				fst[path].VerVect[tnt.me]=tnt.Tree.LogicalTime
				//delete(fst[path].Children, child)
			}
		}
	}
}
