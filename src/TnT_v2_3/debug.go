package TnT_v2_3

import (
	"fmt"
)

func spaces(depth int) {
	for i:=0; i<depth; i++ {
		fmt.Printf(" |")
	}
	fmt.Printf(" |---- ")
}

func (tnt *TnTServer) ParseTree(path string, depth int) {
	fst := tnt.Tree.MyTree

	if _, exists := fst[path]; exists {
		spaces(depth)
		fmt.Println(tnt.me, path, ":	", fst[path].LastModTime, "	", fst[path].VerVect, "	", fst[path].SyncVect, fst[path].Creator, fst[path].CreationTime)
		if fst[path].IsDir {
			for child, _ := range fst[path].Children {
				tnt.ParseTree(child, depth+1)
			}
		}
	}
}

func (tnt *TnTServer) PrintTmp() {
	DelFiles := tnt.Tree.DelFiles
	NewFiles := tnt.Tree.NewFiles

	// Delete files
	for i:=0; i<len(DelFiles); i++ {
		fmt.Println("Del:", DelFiles[i])
	}

	// Update files
	for i:=0; i<len(NewFiles); i++ {
		fmt.Println("New:", NewFiles[i])
	}
}
