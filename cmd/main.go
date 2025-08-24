package main

import (
	"fmt"
	"github.com/sbelectronics/rmxtool/pkg/rmximage"
	"github.com/spf13/cobra"
	"os"
	"path"
	"strconv"
)

var (
	checkErrors    int
	quiet          bool
	byteSwap       bool
	contig         bool
	imageFileName  string
	outputFileName string
	rmxDirectory   string
	destName       string
	rootCmd        = &cobra.Command{
		Use:   "rmxtool",
		Short: "Tool for modifying iRMX disk images",
	}

	dumpCmd = &cobra.Command{
		Use:   "dump",
		Short: "Dump structures to stdout",
		Run:   Dump,
	}

	statCmd = &cobra.Command{
		Use:   "stat",
		Short: "Get file stat",
		Run:   Stat,
	}

	dirCmd = &cobra.Command{
		Use:   "dir",
		Short: "List directory contents",
		Run:   Dir,
	}

	chkdskCmd = &cobra.Command{
		Use:   "chkdsk",
		Short: "check Disk",
		Run:   CheckDisk,
	}

	getCmd = &cobra.Command{
		Use:   "get",
		Short: "Get file from image to local disk",
		Run:   Get,
	}

	putCmd = &cobra.Command{
		Use:   "put",
		Short: "Put from local disk to image",
		Run:   Put,
	}

	deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete file from imagfe",
		Run:   Delete,
	}

	mkdirCmd = &cobra.Command{
		Use:   "mkdir",
		Short: "create directory",
		Run:   Mkdir,
	}

	wipeCmd = &cobra.Command{
		Use:   "wipe",
		Short: "Delete all files on the disk image",
		Run:   Wipe,
	}

	freeCmd = &cobra.Command{
		Use:   "free",
		Short: "Print free blocks",
		Run:   Free,
	}

	getTreeCmd = &cobra.Command{
		Use:   "gettree",
		Short: "Get the entire disk tree",
		Run:   GetTree,
	}

	incFnodeCmd = &cobra.Command{
		Use:   "incfnode",
		Short: "Increase the number of FNodes in the image",
		Run:   IncFnode,
	}
)

func FatalErrCheck(err error) {
	if err != nil {
		fmt.Println("Fatal error:", err)
		os.Exit(-1)
	}
}

func Infof(format string, args ...interface{}) {
	if quiet {
		return
	}
	fmt.Printf(format, args...)
}

func Dump(cmd *cobra.Command, args []string) {
	r := rmximage.NewRMXImage()
	err := r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	ivl, err := r.GetIsoVolumeLabel()
	FatalErrCheck(err)
	ivl.Print()

	fmt.Println("")

	vl, err := r.GetVolumeLabel()
	FatalErrCheck(err)
	vl.Print()

	for i := 0; i < int(vl.MaxFnode); i++ {
		fnode, err := r.GetFNode(i)
		if err != nil {
			fmt.Println("Error getting FNode:", err)
			os.Exit(-1)
		}
		if fnode.IsAllocated() {
			fmt.Println("")
			fmt.Printf("---- FNode %d ----\n", i)
			fnode.Print()
		}
	}

	fmt.Println("\nVol Map:")

	vm, err := r.GetVolMap()
	FatalErrCheck(err)
	vm.Print()

	fmt.Println("\nFNode Map:")

	fm, err := r.GetFNodeMap()
	FatalErrCheck(err)

	fm.Print()

	fmt.Println("")

	dirFNode, err := r.GetRootDirectory()
	FatalErrCheck(err)

	dirList, err := r.GetDirectory(dirFNode)
	FatalErrCheck(err)

	dirList.Print()
}

func Stat(cmd *cobra.Command, args []string) {
	r := rmximage.NewRMXImage()
	err := r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	if len(args) != 1 {
		fmt.Printf("Arguments required: <filename>\n")
		os.Exit(-1)
	}

	fnode, err := r.Lookup(nil, args[0])
	FatalErrCheck(err)

	fnode.Print()

	_, err = r.ReadFile(fnode)
	FatalErrCheck(err)

	fmt.Printf("Indirect Blocks: ")
	for _, ib := range fnode.AllIndirectBlocks {
		fmt.Printf(" %d", ib)
	}
	fmt.Println()

	fmt.Printf("Blocks: ")
	for _, b := range fnode.AllDataBlocks {
		fmt.Printf(" %d", b)
	}
	fmt.Println()
}

func Dir(cmd *cobra.Command, args []string) {
	r := rmximage.NewRMXImage()
	err := r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	dirName := ""
	if len(args) != 0 {
		dirName = args[0]
	}

	fnode, err := r.Lookup(nil, dirName)
	FatalErrCheck(err)

	dirList, err := r.GetDirectory(fnode)
	FatalErrCheck(err)

	dirList.PrintLong()
}

func Get(cmd *cobra.Command, args []string) {
	r := rmximage.NewRMXImage()
	err := r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	for _, arg := range args {
		fnode, err := r.Lookup(nil, arg)
		FatalErrCheck(err)

		data, err := r.ReadFile(fnode)
		FatalErrCheck(err)

		if fnode.Name == "" {
			fmt.Printf("You have encountered the man with no name. Run.\n")
			os.Exit(-1)
		}

		if outputFileName == "" {
			outputFileName = fnode.Name
		}

		var f *os.File
		if outputFileName == "-" {
			outputFileName = "stdout"
			f = os.Stdout
		} else {
			f, err = os.OpenFile(outputFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			FatalErrCheck(err)
			defer func() {
				err := f.Close()
				FatalErrCheck(err)
			}()
		}

		n, err := f.Write(data)
		FatalErrCheck(err)

		if n != len(data) {
			fmt.Printf("Error: Only %d bytes out of %d were written to %s\n", n, len(data), fnode.Name)
			os.Exit(-1)
		}

		Infof("Wrote %d bytes to %s\n", len(data), outputFileName)
	}
}

func GetParentDir(r *rmximage.RMXImage, dirName string) (*rmximage.FNode, error) {
	var dirFNode *rmximage.FNode
	if dirName != "" && dirName != "." {
		var err error
		dirFNode, err = r.Lookup(nil, dirName)
		if err != nil {
			return nil, err
		}

		if !dirFNode.IsDirectory() {
			return nil, fmt.Errorf("specified directory %s is not a directory", dirName)
		}
		return dirFNode, nil
	} else {
		var err error
		dirFNode, err = r.GetRootDirectory()
		return dirFNode, err
	}
}

func Put(cmd *cobra.Command, args []string) {
	r := rmximage.NewRMXImage()
	err := r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	for _, arg := range args {
		if _, err := os.Stat(arg); os.IsNotExist(err) {
			fmt.Printf("File %s does not exist.\n", arg)
			os.Exit(-1)
		}

		pathName := arg
		fileName := path.Base(pathName)

		if destName != "" {
			fileName = destName
		}

		data, err := os.ReadFile(pathName)
		FatalErrCheck(err)

		dirFNode, err := GetParentDir(r, rmxDirectory)
		FatalErrCheck(err)

		fnode, err := r.Lookup(dirFNode, fileName)
		if err == nil {
			// The file already exists
			fmt.Printf("Deleting file %s in directory %s so we can re-PUT it\n", fileName, dirFNode.Name)
			err = r.DeleteFNode(fnode)
			FatalErrCheck(err)
		}

		fnode, err = r.PutFile(dirFNode, fileName, data, contig)
		FatalErrCheck(err)

		Infof("Stored %d bytes to FNode %d (%s)\n", len(data), fnode.Number, fnode.Name)
	}

	err = r.Save()
	FatalErrCheck(err)
}

func Delete(cmd *cobra.Command, args []string) {
	r := rmximage.NewRMXImage()
	err := r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	for _, arg := range args {
		fnode, err := r.Lookup(nil, arg)
		FatalErrCheck(err)

		err = r.DeleteFNode(fnode)
		FatalErrCheck(err)
	}

	err = r.Save()
	FatalErrCheck(err)
}

func Mkdir(cmd *cobra.Command, args []string) {
	r := rmximage.NewRMXImage()
	err := r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	for _, arg := range args {
		dirName := path.Dir(arg)
		baseName := path.Base(arg)

		dirFNode, err := GetParentDir(r, dirName)
		FatalErrCheck(err)

		_, err = r.Mkdir(dirFNode, baseName)
		FatalErrCheck(err)
	}

	err = r.Save()
	FatalErrCheck(err)
}

func WipeFNode(fnode *rmximage.FNode) error {
	if fnode.IsDirectory() {
		dirList, err := fnode.Image.GetDirectory(fnode)
		if err != nil {
			return err
		}
		for _, entry := range dirList.Entries {
			if entry.FNode != 0 {
				childFNode, err := fnode.Image.GetFNode(int(entry.FNode))
				childFNode.Directory = dirList
				childFNode.Name = entry.Name
				//fmt.Printf("Unlink: %+v\n", childFNode)
				if err != nil {
					return err
				}
				err = WipeFNode(childFNode)
				if err != nil {
					return err
				}
			}
		}
	}
	if fnode.Number > 6 {
		err := fnode.Image.DeleteFNode(fnode)
		if err != nil {
			return err
		}
	}
	return nil
}

func Wipe(cmd *cobra.Command, args []string) {
	r := rmximage.NewRMXImage()
	err := r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	rootDir, err := r.GetRootDirectory()
	FatalErrCheck(err)

	err = WipeFNode(rootDir)
	FatalErrCheck(err)

	err = r.Save()
	FatalErrCheck(err)
}

func CheckDisk(cmd *cobra.Command, args []string) {
	checkErrors = 0
	c := &Checker{}
	c.CheckDisk1()
	if checkErrors > 0 {
		fmt.Printf("Disk check completed with %d errors.\n", checkErrors)
		os.Exit(1)
	} else {
		Infof("Disk check completed successfully, no errors found.\n")
	}
}

func Free(cmd *cobra.Command, args []string) {
	r := rmximage.NewRMXImage()
	err := r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	freeBlocks := 0
	freeFNodes := 0

	volMap, err := r.GetVolMap()
	FatalErrCheck(err)
	for i := 0; i < int(volMap.GetNumBits()); i++ {
		if !volMap.IsAlloc(i) {
			freeBlocks += 1
		}
	}

	fnodeMap, err := r.GetFNodeMap()
	FatalErrCheck(err)
	for i := 0; i < int(fnodeMap.GetNumBits()); i++ {
		if !fnodeMap.IsAlloc(i) {
			freeFNodes += 1
		}
	}

	fmt.Printf("Free blocks: %d\n", freeBlocks)
	fmt.Printf("Free FNodes: %d\n", freeFNodes)
}

func GetTree(cmd *cobra.Command, args []string) {
	r := rmximage.NewRMXImage()
	err := r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	vl, err := r.GetVolumeLabel()
	FatalErrCheck(err)

	err = GetDirFNode(r, int(vl.RootFnode), "")
	FatalErrCheck(err)
}

func GetDirFNode(r *rmximage.RMXImage, fnodeNumber int, pathName string) error {
	fnode, err := r.GetFNode(fnodeNumber)
	if err != nil {
		return fmt.Errorf("  Error getting FNode %d: %v\n", fnodeNumber, err)
	}
	if !fnode.IsAllocated() {
		return fmt.Errorf("  Error: FNode %d is not allocated.\n", fnodeNumber)
	}
	data, err := r.ReadFile(fnode)
	if err != nil {
		return fmt.Errorf("  Error reading file for FNode %d: %v\n", fnodeNumber, err)
	}
	if fnode.IsDirectory() {
		dirList, err := r.GetDirectory(fnode)
		if err != nil {
			return fmt.Errorf("Error getting directory: %v\n", err)
		}
		fmt.Printf("Processing dir %s\n", pathName)
		for _, entry := range dirList.Entries {
			var newPathName string
			if pathName == "" {
				newPathName = entry.Name
			} else {
				newPathName = path.Join(pathName, entry.Name)
			}
			if entry.FNode != 0 {
				GetDirFNode(r, int(entry.FNode), newPathName)
			}
		}
	} else if fnode.FType != rmximage.TypeData {
		if !quiet {
			fmt.Printf("Skipping file %s\n", pathName)
		}
	} else {
		if !quiet {
			fmt.Printf("Processing file %s\n", pathName)
		}
		dir := path.Dir(pathName)
		if dir != "" && dir != "." {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				return fmt.Errorf("Error creating directory %s: %v", dir, err)
			}
		}
		f, err := os.OpenFile(pathName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("Error opening output file %s: %v", outputFileName, err)
		}
		defer func() {
			err := f.Close()
			FatalErrCheck(err)
		}()
		_, err = f.Write(data)
		FatalErrCheck(err)
	}

	return nil
}

func IncFnode(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Printf("Usage: %s <fnode count>\n", cmd.Use)
		os.Exit(-1)
	}
	newFnodeCount, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Printf("Invalid fnode count: %s\n", args[0])
		os.Exit(-1)
	}

	r := rmximage.NewRMXImage()
	err = r.Load(imageFileName, byteSwap)
	FatalErrCheck(err)

	vl, err := r.GetVolumeLabel()
	FatalErrCheck(err)

	if newFnodeCount <= int(vl.MaxFnode) {
		fmt.Printf("FNode count must be greater than current max FNode count (%d)\n", vl.MaxFnode)
		os.Exit(-1)
	}

	fnode, err := r.GetFNode(0)
	FatalErrCheck(err)

	data, err := r.ReadFile(fnode)
	FatalErrCheck(err)

	err = r.TruncateFNode(fnode)
	FatalErrCheck(err)

	newSize := uint32(newFnodeCount) * uint32(vl.FnodeSize)
	padding := make([]byte, newSize-uint32(len(data)))
	data = append(data, padding...)

	err = r.PutData(fnode, data, true) // side-effect will write fnode to old location
	FatalErrCheck(err)

	// update the volume label, do this before fnode.Update() so it writes to the
	// correcy place.
	origMaxFnode := vl.MaxFnode
	vl.FnodeStart = uint32(fnode.Pointers[0].BlockPointer) * uint32(vl.Gran)
	vl.MaxFnode = uint16(newFnodeCount)
	err = vl.Update()
	FatalErrCheck(err)

	err = fnode.Update()
	FatalErrCheck(err)

	// make the fnode map bigger

	fmfnode, err := r.GetFNode(2) // FNodeMap
	FatalErrCheck(err)

	fmfnode.TotalSize = uint32(newFnodeCount+7) / uint32(8)

	err = fmfnode.Update()
	FatalErrCheck(err)

	fm, err := r.GetFNodeMap()
	FatalErrCheck(err)
	for i := origMaxFnode; i < uint16(newFnodeCount); i++ {
		fm.SetAlloc(int(i), false)
	}

	err = fm.Update()
	FatalErrCheck(err)

	err = r.Save()
	FatalErrCheck(err)
}

func main() {
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Hide nonessential output")
	rootCmd.PersistentFlags().BoolVarP(&byteSwap, "byteswap", "b", false, "Swap low and high bytes")
	rootCmd.PersistentFlags().StringVarP(&imageFileName, "filename", "f", "test.img", "RMX image file to use")
	rootCmd.AddCommand(dumpCmd)
	rootCmd.AddCommand(statCmd)
	rootCmd.AddCommand(dirCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(putCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(mkdirCmd)
	rootCmd.AddCommand(wipeCmd)
	rootCmd.AddCommand(chkdskCmd)
	rootCmd.AddCommand(freeCmd)
	rootCmd.AddCommand(getTreeCmd)
	rootCmd.AddCommand(incFnodeCmd)

	getCmd.PersistentFlags().StringVarP(&outputFileName, "output", "o", "", "output filename")
	putCmd.PersistentFlags().StringVarP(&rmxDirectory, "directory", "d", "", "parent directory to use in RMX image")
	putCmd.PersistentFlags().StringVarP(&destName, "name", "n", "", "name to use when putting file in RMX image (defaults to basename of file)")
	putCmd.PersistentFlags().BoolVarP(&contig, "contig", "c", false, "Allocate contiguous blocks for the file in the RMX image")

	err := rootCmd.Execute()
	FatalErrCheck(err)
}
