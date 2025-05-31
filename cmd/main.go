package main

import (
	"fmt"
	"github.com/sbelectronics/rmxtool/pkg/rmximage"
	"github.com/spf13/cobra"
	"os"
	"path"
)

var (
	checkErrors    int
	quiet          bool
	byteSwap       bool
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
			defer f.Close()
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
			return nil, fmt.Errorf("Specified directory %s is not a directory.\n", dirName)
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

		fnode, err = r.PutFile(dirFNode, fileName, data)
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

		fmt.Printf("%s %s\n", dirName, baseName)

		dirFNode, err := GetParentDir(r, dirName)
		FatalErrCheck(err)

		fmt.Printf("%d", dirFNode.Number)

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
		fnode.Image.DeleteFNode(fnode)
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

	getCmd.PersistentFlags().StringVarP(&outputFileName, "output", "o", "", "output filename")
	putCmd.PersistentFlags().StringVarP(&rmxDirectory, "directory", "d", "", "parent directory to use in RMX image")
	putCmd.PersistentFlags().StringVarP(&destName, "name", "n", "", "name to use when putting file in RMX image (defaults to basename of file)")

	err := rootCmd.Execute()
	FatalErrCheck(err)
}
