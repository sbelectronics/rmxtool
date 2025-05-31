package main

import (
	"fmt"
	"github.com/sbelectronics/rmxtool/pkg/rmximage"
)

/* CheckDisk is complicated enough that it gets a file all to itself */

type Checker struct {
	r           *rmximage.RMXImage
	Alloc       map[int][]*rmximage.FNode
	AllocFNodes map[int]*rmximage.FNode
}

func (c *Checker) CheckDisk1() {
	c.r = rmximage.NewRMXImage()
	err := c.r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	c.Alloc = map[int][]*rmximage.FNode{}
	c.AllocFNodes = map[int]*rmximage.FNode{}

	vl, err := c.r.GetVolumeLabel()
	if err != nil {
		fmt.Printf("Error getting volume label: %v\n", err)
		checkErrors += 1
		return
	}

	Infof("Volume Name: %s\n", vl.Name)
	c.CheckFNode(0, "FNodeList")
	c.CheckFNode(3, "Unused FNode")
	c.CheckFNode(int(vl.RootFnode), "RootDirectory")

	volMap, err := c.r.GetVolMap()
	if err != nil {
		fmt.Printf("Error getting volume map: %v\n", err)
		checkErrors += 1
		return
	}

	Infof("Reconciling free lists\n")
	for blocknum, fnodes := range c.Alloc {
		if len(fnodes) > 1 {
			fmt.Printf("Block %d is allocated by multiple FNodes:\n", blocknum)
			for _, fnode := range fnodes {
				fmt.Printf("  FNode %d\n", fnode.Number)
			}
			checkErrors += 1
		}
		if !volMap.IsAlloc(blocknum) {
			fmt.Printf("  Block %d is marked as free, but allocated by FNodes:\n", blocknum)
			checkErrors += 1
		}
	}

	for i := 0; i < int(volMap.GetNumBits()); i++ {
		_, isAlloc := c.Alloc[i]
		if volMap.IsAlloc(i) && !isAlloc {
			fmt.Printf("  Block %d is marked as allocated in VolMap but not in allocation map.\n", i)
			checkErrors += 1
		} else if !volMap.IsAlloc(i) && isAlloc {
			fmt.Printf("  Block %d is marked as free in VolMap but allocated in allocation map.\n", i)
			checkErrors += 1
		}
	}

	fnodeMap, err := c.r.GetFNodeMap()
	if err != nil {
		fmt.Printf("Error getting FNode map: %v\n", err)
		checkErrors += 1
		return
	}

	for fnodeIndex := range c.AllocFNodes {
		if !fnodeMap.IsAlloc(fnodeIndex) {
			fmt.Printf("  FNode %d is allocated but not marked in FNode map.\n", fnodeIndex)
			checkErrors += 1
		}
	}

	c.AllocFNodes[3] = &rmximage.FNode{Number: 3}

	for i := 0; i < int(fnodeMap.GetNumBits()); i++ {
		_, isAlloc := c.AllocFNodes[i]
		if fnodeMap.IsAlloc(i) && !isAlloc {
			fmt.Printf("  FNode %d is marked as allocated in FNodeMap but not in allocation map.\n", i)
			checkErrors += 1
		} else if !fnodeMap.IsAlloc(i) && isAlloc {
			fmt.Printf("  FNode %d is marked as free in FNodeMap but allocated in allocation map.\n", i)
			checkErrors += 1
		}
	}
}

func (c *Checker) MarkBlocks(fnode *rmximage.FNode) {
	c.AllocFNodes[fnode.Number] = fnode
	for _, ib := range fnode.AllIndirectBlocks {
		allocEntry, ok := c.Alloc[ib]
		if !ok {
			allocEntry = []*rmximage.FNode{}
		}
		allocEntry = append(allocEntry, fnode)
		c.Alloc[ib] = allocEntry
	}
	for _, b := range fnode.AllDataBlocks {
		allocEntry, ok := c.Alloc[b]
		if !ok {
			allocEntry = []*rmximage.FNode{}
		}
		allocEntry = append(allocEntry, fnode)
		c.Alloc[b] = allocEntry
	}
}

func (c *Checker) CheckFNode(fnodeNumber int, name string) {
	Infof("  Checking fnode %s (#%d)\n", name, fnodeNumber)
	fnode, err := c.r.GetFNode(fnodeNumber)
	if err != nil {
		fmt.Printf("  Error getting FNode %d: %v\n", fnodeNumber, err)
		checkErrors += 1
		return // stop looking at this fnode
	}
	if !fnode.IsAllocated() {
		fmt.Printf("  Error: FNode %d is not allocated.\n", fnodeNumber)
		checkErrors += 1
	}
	_, err = c.r.ReadFile(fnode)
	if err != nil {
		fmt.Printf("  Error reading file for FNode %d: %v\n", fnodeNumber, err)
		checkErrors += 1
		return // stop looking at this fnode
	}
	c.MarkBlocks(fnode)
	if fnode.IsDirectory() {
		Infof("Checking directory %s\n", name)
		c.CheckDir(fnode)
	}
}

func (c *Checker) CheckDir(dir *rmximage.FNode) {
	dirList, err := c.r.GetDirectory(dir)
	if err != nil {
		fmt.Printf("Error getting directory: %v\n", err)
		checkErrors += 1
		return
	}
	for _, entry := range dirList.Entries {
		if entry.FNode != 0 {
			c.CheckFNode(int(entry.FNode), entry.Name)
		}
	}
}
