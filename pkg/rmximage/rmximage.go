package rmximage

import (
	"encoding/binary"
	"fmt"
	"github.com/sbelectronics/rmxtool/pkg/imd"
	"os"
	"strings"
)

const (
	/* Types */
	TypeFNode     = 0
	TypeVolMap    = 1
	TypeFNodeMap  = 2
	TypeAccount   = 3
	TypeBadBlock  = 4
	TypeDirectory = 6
	TypeData      = 8
	TypeVolLabel  = 9
	/* Flags */
	Allocated  = 1
	LongFile   = 2
	Primary    = 4
	Unmodified = 32
	NoDelete   = 64
	/* accessors */
	AccessDelete = 1
	AccessRead   = 2
	AccessAppend = 4
	AccessUpdate = 8
	AccessAll    = (AccessDelete | AccessRead | AccessAppend | AccessUpdate)

	/* number of pointers in fnode */
	NumPointers = 8
)

type RMXImage struct {
	contents []byte
	byteSwap bool
	fileName string
	im       *imd.ImageDisk // if the image is loaded from an IMD file, this will be set
}

type IsoVolumeLabel struct {
	LabelId string // offset 0
	// reserved - offset 3
	Name  string // offset 0+3+1 = 4
	Struc string // offset 4+6 = 10
	// reserved - offset 11
	Side int // offset 11+60 = 71
	// reserved - offset 72
	Interleave int // offset 72+4 = 76
	// reserved - offset 78
	IsoVersion int // offset 79
	slice      []byte
}

var TypeNames = map[int]string{
	TypeFNode:     "FNode",
	TypeVolMap:    "VolMap",
	TypeFNodeMap:  "FNodeMap",
	TypeAccount:   "Account",
	TypeBadBlock:  "BadBlock",
	TypeDirectory: "Directory",
	TypeData:      "Data",
	TypeVolLabel:  "VolLabel",
}

var AccessorNames = map[int]string{
	AccessDelete: "Delete",
	AccessRead:   "Read",
	AccessAppend: "Append",
	AccessUpdate: "Update",
}

func getStr(data []byte) string {
	end := 0
	for i, b := range data {
		if b == 0 {
			end = i
			break
		}
	}
	if end == 0 {
		// no null terminator found, return the whole string
		return string(data)
	}
	return string(data[:end])
}

func putStr(data []byte, str string, length int) {
	if len(str) > length {
		str = str[:length]
	}
	copy(data[:length], str)
	for i := len(str); i < length; i++ {
		data[i] = 0 // fill the rest with zeros
	}
}

func accessStr(access int) string {
	astr := ""
	if access&AccessDelete != 0 {
		astr = astr + "D"
	}
	if access&AccessRead != 0 {
		astr = astr + "R"
	}
	if access&AccessAppend != 0 {
		astr = astr + "A"
	}
	if access&AccessUpdate != 0 {
		astr = astr + "U"
	}

	return astr
}

func (v *IsoVolumeLabel) Deserialize(data []byte) {
	v.LabelId = getStr(data[0:3])
	v.Name = getStr(data[4:10])
	v.Struc = string(data[10])
	v.Side = int(data[71]) - '0'
	v.Interleave = (int(data[76]-'0') * 10) + int(data[77]-'0')
	v.IsoVersion = int(data[79] - '0')
	v.slice = data
}

func (v *IsoVolumeLabel) Serialize(data []byte) {
	copy(data[0:3], v.LabelId)
	copy(data[4:10], v.Name)
	data[10] = v.Struc[0]
	data[71] = byte(v.Side + '0')
	data[76] = byte((v.Interleave / 10) + '0')
	data[77] = byte((v.Interleave % 10) + '0')
	data[79] = byte(v.IsoVersion + '0')
}

func (v *IsoVolumeLabel) Update() {
	v.Serialize(v.slice)
}

func (v *IsoVolumeLabel) Print() {
	fmt.Printf("LabelId: %s\n", v.LabelId)
	fmt.Printf("Name: %s\n", v.Name)
	fmt.Printf("Struc: %s\n", v.Struc)
	fmt.Printf("Side: %d\n", v.Side)
	fmt.Printf("Interleave: %d\n", v.Interleave)
	fmt.Printf("IsoVersion: %d\n", v.IsoVersion)
}

type RmxVolumeLabel struct {
	Name       string
	Fill       uint8
	Driver     uint8
	Gran       uint16
	Size       uint32
	MaxFnode   uint16
	FnodeStart uint32
	FnodeSize  uint16
	RootFnode  uint16

	Image *RMXImage // reference to the RMXImage this FNode belongs to, set by GetFNode()
}

func (v *RmxVolumeLabel) Deserialize(data []byte) {
	v.Name = getStr(data[0:10])
	v.Fill = data[10]
	v.Driver = data[11]
	v.Gran = binary.LittleEndian.Uint16(data[12:14])
	v.Size = binary.LittleEndian.Uint32(data[14:18])
	v.MaxFnode = binary.LittleEndian.Uint16(data[18:20])
	v.FnodeStart = binary.LittleEndian.Uint32(data[20:24])
	v.FnodeSize = binary.LittleEndian.Uint16(data[24:26])
	v.RootFnode = binary.LittleEndian.Uint16(data[26:28])
}

func (v *RmxVolumeLabel) Serialize(data []byte) {
	putStr(data[0:10], v.Name, 10)
	data[10] = v.Fill
	data[11] = v.Driver
	binary.LittleEndian.PutUint16(data[12:14], v.Gran)
	binary.LittleEndian.PutUint32(data[14:18], v.Size)
	binary.LittleEndian.PutUint16(data[18:20], v.MaxFnode)
	binary.LittleEndian.PutUint32(data[20:24], v.FnodeStart)
	binary.LittleEndian.PutUint16(data[24:26], v.FnodeSize)
	binary.LittleEndian.PutUint16(data[26:28], v.RootFnode)
}

func (v *RmxVolumeLabel) Print() {
	fmt.Printf("Name: %s\n", v.Name)
	fmt.Printf("Fill: %d\n", v.Fill)
	fmt.Printf("Driver: %d\n", v.Driver)
	fmt.Printf("Granularity: %d\n", v.Gran)
	fmt.Printf("Size: %d\n", v.Size)
	fmt.Printf("Max Fnode: %d\n", v.MaxFnode)
	fmt.Printf("Fnode Start: %d\n", v.FnodeStart)
	fmt.Printf("Fnode Size: %d\n", v.FnodeSize)
	fmt.Printf("Root Fnode: %d\n", v.RootFnode)
}

func (v *RmxVolumeLabel) Update() error {
	if v.Image == nil {
		return fmt.Errorf("Volume Label does not have an associated RMXImage")
	}
	return v.Image.PutVolumeLabel(v)
}

type Pointer struct {
	NumBlocks    uint16
	BlockPointer uint32 /* actually uint24 */
}

type FNode struct {
	Flags       uint16
	FType       uint8
	Gran        uint8
	Owner       uint16
	CreateTime  uint32
	AccessTime  uint32
	ModifyTime  uint32
	TotalSize   uint32
	TotalBlocks uint32
	Pointers    [NumPointers]Pointer
	ThisSize    uint32
	ReservedA   uint16
	ReservedB   uint16
	IDCount     uint16
	Accessor    [3]struct {
		Access uint8
		Id     uint16
	}
	Parent            uint16
	Name              string     // if Fnode is the result of Lookup() call
	Directory         *Directory // the directory this FNode belongs to, set by Lookup()
	Number            int        // Set in GetFNode()
	Image             *RMXImage  // reference to the RMXImage this FNode belongs to, set by GetFNode()
	AllDataBlocks     []int      // to make it easy to update in place
	AllIndirectBlocks []int
}

func (f *FNode) Deserialize(data []byte) {
	f.Flags = binary.LittleEndian.Uint16(data[0:2])
	f.FType = data[2]
	f.Gran = data[3]
	f.Owner = binary.LittleEndian.Uint16(data[4:6])
	f.CreateTime = binary.LittleEndian.Uint32(data[6:10])
	f.AccessTime = binary.LittleEndian.Uint32(data[10:14])
	f.ModifyTime = binary.LittleEndian.Uint32(data[14:18])
	f.TotalSize = binary.LittleEndian.Uint32(data[18:22])
	f.TotalBlocks = binary.LittleEndian.Uint32(data[22:26])
	for i := 0; i < NumPointers; i++ {
		f.Pointers[i].NumBlocks = binary.LittleEndian.Uint16(data[26+i*5 : 28+i*5])
		f.Pointers[i].BlockPointer = uint32(data[28+i*5]) + uint32(data[29+i*5])<<8 + uint32(data[30+i*5])<<16
	}
	f.ThisSize = binary.LittleEndian.Uint32(data[66:70])
	f.ReservedA = binary.LittleEndian.Uint16(data[70:72])
	f.ReservedB = binary.LittleEndian.Uint16(data[72:74])
	f.IDCount = binary.LittleEndian.Uint16(data[74:76])
	for i := 0; i < 3; i++ {
		f.Accessor[i].Access = data[76+i*3]
		f.Accessor[i].Id = binary.LittleEndian.Uint16(data[77+i*3 : 79+i*3])
	}
	f.Parent = binary.LittleEndian.Uint16(data[85:87])
}

func (f *FNode) Serialize(data []byte) {
	binary.LittleEndian.PutUint16(data[0:2], f.Flags)
	data[2] = f.FType
	data[3] = f.Gran
	binary.LittleEndian.PutUint16(data[4:6], f.Owner)
	binary.LittleEndian.PutUint32(data[6:10], f.CreateTime)
	binary.LittleEndian.PutUint32(data[10:14], f.AccessTime)
	binary.LittleEndian.PutUint32(data[14:18], f.ModifyTime)
	binary.LittleEndian.PutUint32(data[18:22], f.TotalSize)
	binary.LittleEndian.PutUint32(data[22:26], f.TotalBlocks)
	for i := 0; i < NumPointers; i++ {
		binary.LittleEndian.PutUint16(data[26+i*5:28+i*5], f.Pointers[i].NumBlocks)
		data[28+i*5] = byte(f.Pointers[i].BlockPointer)
		data[29+i*5] = byte(f.Pointers[i].BlockPointer >> 8)
		data[30+i*5] = byte(f.Pointers[i].BlockPointer >> 16)
	}
	binary.LittleEndian.PutUint32(data[66:70], f.ThisSize)
	binary.LittleEndian.PutUint16(data[70:72], f.ReservedA)
	binary.LittleEndian.PutUint16(data[72:74], f.ReservedB)
	binary.LittleEndian.PutUint16(data[74:76], f.IDCount)
	for i := 0; i < 3; i++ {
		data[76+i*3] = f.Accessor[i].Access
		binary.LittleEndian.PutUint16(data[77+i*3:79+i*3], f.Accessor[i].Id)
	}
	binary.LittleEndian.PutUint16(data[85:87], f.Parent)
}

func (f *FNode) Print() {
	fmt.Printf("Flags: %d", f.Flags)
	if f.IsAllocated() {
		fmt.Print(" ALLOC")
	}
	if f.IsLong() {
		fmt.Print(" LONG")
	}
	fmt.Println()

	fmt.Printf("FType: %d", f.FType)
	typeName, ok := TypeNames[int(f.FType)]
	if ok {
		fmt.Printf(" (%s)\n", typeName)
	} else {
		fmt.Printf(" (Unknown Type %d)\n", f.FType)
	}

	fmt.Printf("Gran: %d\n", f.Gran)
	fmt.Printf("Owner: %d\n", f.Owner)
	fmt.Printf("CreateTime: %d\n", f.CreateTime)
	fmt.Printf("AccessTime: %d\n", f.AccessTime)
	fmt.Printf("ModifyTime: %d\n", f.ModifyTime)
	fmt.Printf("TotalSize: %d\n", f.TotalSize)
	fmt.Printf("TotalBlocks: %d\n", f.TotalBlocks)
	for i, p := range f.Pointers {
		fmt.Printf("Pointer[%d]: NumBlocks=%d, BlockPointer=%d\n", i, p.NumBlocks, p.BlockPointer)
	}
	fmt.Printf("ThisSize: %d\n", f.ThisSize)
	fmt.Printf("ReservedA: %d\n", f.ReservedA)
	fmt.Printf("ReservedB: %d\n", f.ReservedB)
	fmt.Printf("IDCount: %d\n", f.IDCount)
	for i, acc := range f.Accessor {
		fmt.Printf("Accessor[%d]: Access=%d, Id=%d\n", i, acc.Access, acc.Id)
	}
	fmt.Printf("Parent: %d\n", f.Parent)
}

func (f *FNode) IsAllocated() bool {
	return f.Flags&Allocated != 0
}

func (f *FNode) IsLong() bool {
	return f.Flags&LongFile != 0
}

func (f *FNode) IsPrimary() bool {
	return f.Flags&Primary != 0
}

func (f *FNode) IsUnmodified() bool {
	return f.Flags&Unmodified != 0
}

func (f *FNode) IsNoDelete() bool {
	return f.Flags&NoDelete != 0
}

func (f *FNode) IsDirectory() bool {
	return f.FType == TypeDirectory
}

func (f *FNode) SetAlloc(alloc bool) {
	if alloc {
		f.Flags |= Allocated
	} else {
		f.Flags &^= Allocated
	}
}

func (f *FNode) appendAllDataBlocks(numBlocks int, blockPointer int) {
	if f.AllDataBlocks == nil {
		f.AllDataBlocks = []int{}
	}
	for i := 0; i < int(numBlocks); i++ {
		f.AllDataBlocks = append(f.AllDataBlocks, blockPointer+i)
	}
}

func (f *FNode) appendAllIndirectBlocks(numBlocks int, blockPointer int) {
	if f.AllIndirectBlocks == nil {
		f.AllIndirectBlocks = []int{}
	}
	for i := 0; i < int(numBlocks); i++ {
		f.AllIndirectBlocks = append(f.AllIndirectBlocks, blockPointer+i)
	}
}

func (f *FNode) UpdateDataInPlace(data []byte) error {
	if f.Image == nil {
		return fmt.Errorf("FNode does not have an associated RMXImage")
	}
	vl, err := f.Image.GetVolumeLabel()
	if err != nil {
		return err
	}
	index := 0
	for len(data) > 0 {
		blkSize := min(len(data), int(vl.Gran))
		blk := data[:blkSize]
		blkNum := f.AllDataBlocks[index]
		start := blkNum * int(vl.Gran)
		end := start + blkSize
		copy(f.Image.contents[start:end], blk)

		data = data[blkSize:]
		index += 1
	}
	return nil
}

func (f *FNode) GetFreePointer() (int, error) {
	for i := 0; i < NumPointers; i++ {
		if f.Pointers[i].NumBlocks == 0 {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no free pointer available in FNode")
}

func (f *FNode) Expand() error {
	if f.Image == nil {
		return fmt.Errorf("FNode does not have an associated RMXImage")
	}
	vl, err := f.Image.GetVolumeLabel()
	if err != nil {
		return err
	}

	volMap, err := f.Image.GetVolMap()
	if err != nil {
		return err
	}

	freeBlock, err := volMap.NextFree()
	if err != nil {
		return err
	}

	volMap.SetAlloc(freeBlock, true)
	err = volMap.Update()
	if err != nil {
		return err
	}

	ptr, err := f.GetFreePointer()
	if err != nil {
		return err
	}

	f.Pointers[ptr].NumBlocks = 1
	f.Pointers[ptr].BlockPointer = uint32(freeBlock)
	f.ThisSize += uint32(vl.Gran)
	f.TotalBlocks += 1
	f.AllDataBlocks = append(f.AllDataBlocks, freeBlock)
	err = f.Update()
	if err != nil {
		return err
	}

	return nil
}

func (f *FNode) AddAccessor(access int, id int) error {
	if f.IDCount >= 3 {
		return fmt.Errorf("FNode already has maximum number of accessors (3)")
	}
	f.Accessor[f.IDCount].Access = uint8(access)
	f.Accessor[f.IDCount].Id = uint16(id)
	f.IDCount += 1
	return nil
}

func (f *FNode) Update() error {
	if f.Image == nil {
		return fmt.Errorf("FNode does not have an associated RMXImage")
	}
	return f.Image.PutFNode(f.Number, f)
}

type DirEntry struct {
	FNode uint16
	Name  string
}

type Directory struct {
	Entries []DirEntry
	image   *RMXImage
	fnode   *FNode // the FNode this Directory belongs to, set by GetDirectory()
}

func (d *Directory) Deserialize(data []byte, length int) {
	d.Entries = []DirEntry{}
	offset := 0
	for length >= 16 {
		entry := DirEntry{}
		entry.FNode = binary.LittleEndian.Uint16(data[offset+0 : offset+2])
		entry.Name = getStr(data[offset+2 : offset+16])
		offset += 16
		length -= 16
		d.Entries = append(d.Entries, entry)
	}
}

func (d *Directory) Serialize(data []byte) {
	offset := 0
	for _, entry := range d.Entries {
		if len(entry.Name) > 14 {
			entry.Name = entry.Name[:14]
		}
		binary.LittleEndian.PutUint16(data[offset+0:offset+2], entry.FNode)
		putStr(data[offset+2:offset+16], entry.Name, 14)
		offset += 16
	}
	for i := offset; i < len(data); i++ {
		data[i] = 0 // fill the rest with zeros
	}
}

func (d *Directory) Print() {
	for _, entry := range d.Entries {
		if entry.FNode != 0 {
			fmt.Printf("%-15s %8d\n", entry.Name, entry.FNode)
		}
	}
}

func (d *Directory) PrintLong() {
	fmt.Printf("%-15s %8s %8s %-12s %s %s\n", "Name", "FNode", "Size", " Type", "Flags", " Accessors")
	fmt.Printf("%-15s %8s %8s %-12s %s %s\n", "----", "-----", "----", " ----", "-----", " ---------")
	for _, entry := range d.Entries {
		if entry.FNode == 0 {
			continue
		}
		fmt.Printf("%-15s %8d", entry.Name, entry.FNode)
		fnode, err := d.image.GetFNode(int(entry.FNode))
		if err != nil {
			fmt.Printf("ERR\n")
		} else {
			fmt.Printf(" %8d", fnode.TotalSize)

			typeName, ok := TypeNames[int(fnode.FType)]
			if ok {
				fmt.Printf("  %-12s", typeName)
			} else {
				fmt.Printf("  %-12s", "Unknown")
			}

			if fnode.IsAllocated() {
				fmt.Print("A")
			} else {
				fmt.Print(" ")
			}
			if fnode.IsLong() {
				fmt.Print("L")
			} else {
				fmt.Print(" ")
			}
			if fnode.IsPrimary() {
				fmt.Print("P")
			} else {
				fmt.Print(" ")
			}
			if fnode.IsUnmodified() {
				fmt.Print("U")
			} else {
				fmt.Print(" ")
			}
			if fnode.IsNoDelete() {
				fmt.Print("N")
			} else {
				fmt.Print(" ")
			}

			fmt.Printf(" ")
			for i := 0; i < int(fnode.IDCount); i++ {
				accessor := fnode.Accessor[i]
				if accessor.Access != 0 {
					fmt.Printf(" %s:%d", accessStr(int(accessor.Access)), accessor.Id)
				}
			}
		}
		fmt.Println()
	}
}

func (d *Directory) Find(entryName string) (int, error) {
	for i := range d.Entries {
		if strings.EqualFold(d.Entries[i].Name, entryName) && d.Entries[i].FNode != 0 {
			return int(d.Entries[i].FNode), nil
		}
	}
	return 0, fmt.Errorf("entry '%s' not found in directory", entryName)
}

func (d *Directory) Unlink(entryName string) error {
	found := false
	for i := range d.Entries {
		if strings.EqualFold(d.Entries[i].Name, entryName) {
			d.Entries[i].FNode = 0 // Unlink the entry by setting FNode to 0
			found = true
		}
	}
	if !found {
		return fmt.Errorf("entry '%s' not found in directory", entryName)
	}
	return nil
}

func (d *Directory) AddEntry(fnodeIndex int, name string) (*DirEntry, error) {
	if len(name) > 14 {
		name = name[:14] // truncate name to fit in 14 characters
	}
	for i := range d.Entries {
		entry := &d.Entries[i]
		if entry.FNode == 0 {
			entry.FNode = uint16(fnodeIndex)
			entry.Name = name
			return entry, nil
		}
	}
	if d.fnode == nil {
		return nil, fmt.Errorf("Directory does not have an associated FNode")
	}
	if d.fnode.TotalSize+16 > d.fnode.ThisSize {
		err := d.fnode.Expand()
		if err != nil {
			return nil, err
		}
	}

	entry := &DirEntry{FNode: uint16(fnodeIndex), Name: name}
	d.Entries = append(d.Entries, *entry)
	d.fnode.TotalSize += 16
	err := d.fnode.Update()
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (d *Directory) Update() error {
	if d.fnode == nil {
		return fmt.Errorf("Directory does not have an associated FNode")
	}
	data := make([]byte, 16*len(d.Entries))
	d.Serialize(data)
	return d.fnode.UpdateDataInPlace(data)
}

type Bitmap struct {
	data    []byte
	fnode   *FNode
	numBits int
}

func (v *Bitmap) IsAlloc(n int) bool {
	byteIndex := n / 8
	bitIndex := n % 8
	if byteIndex >= len(v.data) {
		return false
	}
	return v.data[byteIndex]&(1<<bitIndex) == 0
}

func (v *Bitmap) SetAlloc(n int, alloc bool) {
	byteIndex := n / 8
	bitIndex := n % 8
	if !alloc {
		v.data[byteIndex] |= (1 << bitIndex)
	} else {
		v.data[byteIndex] &^= (1 << bitIndex)
	}
}

func (v *Bitmap) NextFree() (int, error) {
	for i := 0; i < v.numBits; i++ {
		if !v.IsAlloc(i) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no free bits")
}

func (v *Bitmap) GetFreeRange(count int, contig bool) ([]int, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be greater than 0")
	}

	blocks := []int{}

	for i := 0; i < v.numBits; i++ {
		if !v.IsAlloc(i) {
			blocks = append(blocks, i)
			if len(blocks) == count {
				return blocks, nil
			}
		} else {
			if contig {
				blocks = []int{} // reset if we hit an allocated block
			}
		}
	}
	if contig {
		return nil, fmt.Errorf("no contiguous free ranges for size %d", count)
	} else {
		return nil, fmt.Errorf("not enough free bits for size %d", count)
	}
}

func (v *Bitmap) Print() {
	start := -1
	for i := 0; i < v.numBits; i++ {
		if v.IsAlloc(i) {
			if start < 0 {
				start = i
			}
		} else {
			if start >= 0 {
				fmt.Printf("%d-%d ", start, i-1)
				start = -1
			}
		}
	}
	if start >= 0 {
		fmt.Printf("%d-%d ", start, v.numBits-1)
	}
	fmt.Println()
}

func (v *Bitmap) GetNumBits() int {
	return v.numBits
}

func (v *Bitmap) Update() error {
	if v.fnode == nil {
		return fmt.Errorf("Bitmap does not have an associated FNode")
	}
	return v.fnode.UpdateDataInPlace(v.data)
}

func NewRMXImage() *RMXImage {
	return &RMXImage{}
}

func (r *RMXImage) Load(fileName string, byteSwap bool) error {
	r.fileName = fileName
	r.byteSwap = byteSwap

	var data []byte
	if strings.HasSuffix(fileName, ".imd") || strings.HasSuffix(fileName, ".IMD") {
		im := imd.NewImageDisk()
		err := im.Load(fileName)
		if err != nil {
			return err
		}
		data = im.GetData()

		/*
			err = os.WriteFile("imdtest.img", data, 0644)
			if err != nil {
				return err
			}
		*/
		r.im = im
	} else {
		var err error
		data, err = os.ReadFile(fileName)
		if err != nil {
			return err
		}
	}

	if byteSwap {
		for i := 0; i < len(data); i += 2 {
			if i+1 < len(data) {
				data[i], data[i+1] = data[i+1], data[i]
			}
		}
	}

	r.contents = data
	return nil
}

func (r *RMXImage) Save() error {
	if r.fileName == "" {
		return fmt.Errorf("no file name specified for saving RMXImage")
	}

	var data []byte

	if strings.HasSuffix(r.fileName, ".imd") || strings.HasSuffix(r.fileName, ".IMD") {
		var err error
		r.im.SetData(r.contents)
		data, err = r.im.GetIMD()
		if err != nil {
			return fmt.Errorf("failed to get IMD data: %w", err)
		}
	} else {
		data = r.contents
	}

	if r.byteSwap {
		for i := 0; i < len(r.contents); i += 2 {
			if i+1 < len(r.contents) {
				data[i], data[i+1] = data[i+1], data[i]
			}
		}
	}

	err := os.Remove(r.fileName)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete existing file: %w", err)
	}

	return os.WriteFile(r.fileName, data, 0644)
}

func (r *RMXImage) GetIsoVolumeLabel() (*IsoVolumeLabel, error) {
	if len(r.contents) < 896 {
		return nil, os.ErrInvalid
	}
	label := &IsoVolumeLabel{}
	label.Deserialize(r.contents[768:])
	return label, nil
}

func (r *RMXImage) GetVolumeLabel() (*RmxVolumeLabel, error) {
	if len(r.contents) < 512 {
		return nil, os.ErrInvalid
	}
	label := &RmxVolumeLabel{}
	label.Deserialize(r.contents[384:])
	label.Image = r
	return label, nil
}

func (r *RMXImage) PutVolumeLabel(label *RmxVolumeLabel) error {
	if len(r.contents) < 512 {
		return os.ErrInvalid
	}
	label.Serialize(r.contents[384:])
	return nil
}

func (r *RMXImage) GetFNode(fnodeIndex int) (*FNode, error) {
	vl, err := r.GetVolumeLabel()
	if err != nil {
		return nil, err
	}
	offset := vl.FnodeStart + uint32(fnodeIndex)*uint32(vl.FnodeSize)
	fnode := &FNode{Image: r, Number: fnodeIndex}
	fnode.Deserialize(r.contents[offset : offset+uint32(vl.FnodeSize)])
	return fnode, nil
}

func (r *RMXImage) PutFNode(fnodeIndex int, fnode *FNode) error {
	vl, err := r.GetVolumeLabel()
	if err != nil {
		return err
	}
	offset := vl.FnodeStart + uint32(fnodeIndex)*uint32(vl.FnodeSize)
	fnode.Serialize(r.contents[offset : offset+uint32(vl.FnodeSize)])
	return nil
}

func (r *RMXImage) ReadLongData(fnode *FNode, blockfile []byte, totalBlocks int) ([]byte, int, error) {
	vl, err := r.GetVolumeLabel()
	if err != nil {
		return nil, 0, err
	}
	gran := int(vl.Gran)
	data := []byte{}
	for (len(blockfile) > 0) && (totalBlocks > 0) {
		nblocks := blockfile[0]
		blockPointer := uint32(blockfile[1]) + uint32(blockfile[2])<<8 + uint32(blockfile[3])<<16
		blockfile = blockfile[4:]

		if nblocks == 0 {
			continue
		}

		fnode.appendAllDataBlocks(int(nblocks), int(blockPointer))

		start := uint32(blockPointer) * uint32(vl.Gran)
		end := start + uint32(nblocks)*uint32(gran)
		//fmt.Printf("%d %d %d %d %d\n", totalBlocks, nblocks, blockPointer, start, end)
		data = append(data, r.contents[start:end]...)

		totalBlocks -= int(nblocks)
	}
	return data, totalBlocks, nil
}

func (r *RMXImage) ReadFile(fnode *FNode) ([]byte, error) {
	vl, err := r.GetVolumeLabel()
	if err != nil {
		return nil, err
	}
	gran := int(vl.Gran) * int(fnode.Gran)
	data := []byte{}
	if fnode.IsLong() {
		totalBlocks := int(fnode.TotalBlocks)
		for _, pointer := range fnode.Pointers {
			var err error
			var thisData []byte
			if pointer.NumBlocks == 0 {
				continue
			}
			fnode.appendAllIndirectBlocks(1, int(pointer.BlockPointer))
			totalBlocks -= 1 // gotta count the indirect block too. Assuming can only name 1 indirect block.
			start := pointer.BlockPointer * uint32(vl.Gran)
			end := start + 1*uint32(gran)
			thisData, totalBlocks, err = r.ReadLongData(fnode, r.contents[start:end], totalBlocks)
			if err != nil {
				return nil, err
			}
			data = append(data, thisData...)
		}
	} else {
		for _, pointer := range fnode.Pointers {
			if pointer.NumBlocks == 0 {
				continue
			}
			fnode.appendAllDataBlocks(int(pointer.NumBlocks), int(pointer.BlockPointer))
			//fmt.Printf("%d %d %d\n", pointer.NumBlocks, pointer.BlockPointer, len(r.contents))
			start := uint32(pointer.BlockPointer) * uint32(vl.Gran)
			end := start + uint32(pointer.NumBlocks)*uint32(gran)
			data = append(data, r.contents[start:end]...)
		}
	}
	data = data[:fnode.TotalSize]
	return data, nil
}

func (r *RMXImage) Mknod(dirFNode *FNode, fileName string, ftype int) (*FNode, error) {
	if !dirFNode.IsDirectory() {
		return nil, fmt.Errorf("parent FNode is not a directory")
	}

	fnode := &FNode{
		Name:  fileName,
		Image: r,
		FType: uint8(ftype),
		Flags: Allocated | Primary,
		Gran:  1,
		Owner: uint16(dirFNode.Number),
	}

	err := fnode.AddAccessor(AccessAll, 0) // Root
	if err != nil {
		return nil, err
	}
	err = fnode.AddAccessor(AccessAll, 65535) // World
	if err != nil {
		return nil, err
	}

	fnodeMap, err := r.GetFNodeMap()
	if err != nil {
		return nil, err
	}

	fnode.Number, err = fnodeMap.NextFree()
	if err != nil {
		return nil, err
	}

	fnodeMap.SetAlloc(fnode.Number, true)
	err = fnodeMap.Update()
	if err != nil {
		return nil, err
	}

	err = r.PutFNode(fnode.Number, fnode)
	if err != nil {
		return nil, err
	}

	dirList, err := r.GetDirectory(dirFNode)
	if err != nil {
		return nil, err
	}

	_, err = dirList.AddEntry(fnode.Number, fnode.Name)
	if err != nil {
		return nil, err
	}

	err = dirList.Update()
	if err != nil {
		return nil, err
	}

	return fnode, nil
}

func (r *RMXImage) PutFile(dirFNode *FNode, fileName string, data []byte, contig bool) (*FNode, error) {
	fnode, err := r.Mknod(dirFNode, fileName, TypeData)
	if err != nil {
		return nil, err
	}

	err = r.PutData(fnode, data, contig)
	if err != nil {
		return nil, err
	}

	return fnode, nil
}

func (r *RMXImage) PutData(fnode *FNode, data []byte, contig bool) error {
	vl, err := r.GetVolumeLabel()
	if err != nil {
		return err
	}

	fnode.TotalSize = uint32(len(data))

	// Note that Mknod might affect volmap, so do this after calling mknod
	volMap, err := r.GetVolMap()
	if err != nil {
		return err
	}

	blockCount := (len(data) + int(vl.Gran) - 1) / int(vl.Gran)
	fmt.Printf("Allocating %d blocks for file '%s' gran=%d\n", blockCount, fnode.Name, vl.Gran)
	blockNums, err := volMap.GetFreeRange(blockCount, contig)
	if err != nil {
		return err
	}

	blkList := []Pointer{}

	blockIndex := 0
	start := -1
	last := -1
	for len(data) > 0 {
		blkNum := blockNums[blockIndex]
		//fmt.Printf("Allocating block %d for file '%s'\n", blkNum, fileName)
		if blkNum != last+1 {
			if start != -1 {
				blkList = append(blkList, Pointer{NumBlocks: uint16(last - start + 1), BlockPointer: uint32(start)})
			}
			start = -1
		}
		if start < 0 {
			start = blkNum
		}
		blkSize := min(len(data), int(vl.Gran))
		startAddr := blkNum * int(vl.Gran)
		endAddr := startAddr + blkSize
		copy(r.contents[startAddr:endAddr], data[:blkSize])
		data = data[blkSize:]
		volMap.SetAlloc(blkNum, true) // Mark the block as allocated
		last = blkNum

		fnode.TotalBlocks += 1
		fnode.ThisSize += uint32(vl.Gran)

		blockIndex += 1
	}

	if start != -1 {
		blkList = append(blkList, Pointer{NumBlocks: uint16(last - start + 1), BlockPointer: uint32(start)})
	}

	if len(blkList) > 8 {
		return fmt.Errorf("too many blocks allocated for FNode: %d. Long files not supported yet", len(blkList))
	}

	err = volMap.Update()
	if err != nil {
		return err
	}

	fmt.Printf("XXX %d", fnode.TotalBlocks)
	copy(fnode.Pointers[:], blkList)
	err = fnode.Update()
	if err != nil {
		return err
	}

	return nil
}

func (r *RMXImage) Mkdir(parentFNode *FNode, dirName string) (*FNode, error) {
	// Tt's okay to create a 0-size directory
	// It will get expanded when entries are added

	fnode, err := r.Mknod(parentFNode, dirName, TypeDirectory)
	if err != nil {
		return nil, err
	}

	return fnode, nil
}

func (r *RMXImage) GetDirectory(dirFnode *FNode) (*Directory, error) {
	if !dirFnode.IsDirectory() {
		return nil, fmt.Errorf("FNode is not a directory")
	}
	data, err := r.ReadFile(dirFnode)
	if err != nil {
		return nil, err
	}
	dir := &Directory{image: r, fnode: dirFnode}
	dir.Deserialize(data, len(data))
	return dir, nil
}

func (r *RMXImage) GetRootDirectory() (*FNode, error) {
	vl, err := r.GetVolumeLabel()
	if err != nil {
		return nil, err
	}
	dirFNode, err := r.GetFNode(int(vl.RootFnode))
	if err != nil {
		return nil, err
	}
	return dirFNode, nil
}

func (r *RMXImage) GetVolMap() (*Bitmap, error) {
	vl, err := r.GetVolumeLabel()
	if err != nil {
		return nil, err
	}
	fnode, err := r.GetFNode(1)
	if err != nil {
		return nil, err
	}
	data, err := r.ReadFile(fnode)
	if err != nil {
		return nil, err
	}
	b := &Bitmap{
		data:    data,
		fnode:   fnode,
		numBits: int(vl.Size) / int(vl.Gran),
	}

	return b, nil
}

func (r *RMXImage) GetFNodeMap() (*Bitmap, error) {
	vl, err := r.GetVolumeLabel()
	if err != nil {
		return nil, err
	}
	fnode, err := r.GetFNode(2)
	if err != nil {
		return nil, err
	}
	data, err := r.ReadFile(fnode)
	if err != nil {
		return nil, err
	}
	b := &Bitmap{
		data:    data,
		fnode:   fnode,
		numBits: int(vl.MaxFnode),
	}

	return b, nil
}

func (r *RMXImage) Lookup(dir *FNode, name string) (*FNode, error) {
	name = strings.TrimPrefix(name, "/")

	if dir == nil {
		vl, err := r.GetVolumeLabel()
		if err != nil {
			return nil, err
		}
		dir, err = r.GetFNode(int(vl.RootFnode))
		if err != nil {
			return nil, err
		}
	}

	if name == "" {
		return dir, nil
	}

	dirList, err := r.GetDirectory(dir)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(name, "/", 2)

	fnodeIndex, err := dirList.Find(parts[0])
	if err != nil {
		return nil, err
	}

	fnode, err := r.GetFNode(fnodeIndex)
	if err != nil {
		return nil, err
	}

	fnode.Name = parts[0]
	fnode.Directory = dirList

	if fnode.IsDirectory() {
		if len(parts) > 1 {
			return r.Lookup(fnode, parts[1])
		}
		return fnode, nil
	} else {
		if len(parts) > 1 {
			return nil, fmt.Errorf("file %s is not a directory", parts[0])
		}
		return fnode, nil
	}
}

func (r *RMXImage) DeleteFNode(fnode *FNode) error {
	err := r.TruncateFNode(fnode)
	if err != nil {
		return err
	}

	fnodeMap, err := r.GetFNodeMap()
	if err != nil {
		return err
	}

	fnodeMap.SetAlloc(fnode.Number, false)

	fnode.SetAlloc(false)
	err = fnode.Update()
	if err != nil {
		return err
	}

	err = fnode.Directory.Unlink(fnode.Name)
	if err != nil {
		return err
	}

	err = fnode.Directory.Update()
	if err != nil {
		return err
	}

	err = fnodeMap.Update()
	if err != nil {
		return err
	}

	return nil
}

func (r *RMXImage) TruncateFNode(fnode *FNode) error {
	volMap, err := r.GetVolMap()
	if err != nil {
		return err
	}

	_, err = r.ReadFile(fnode)
	if err != nil {
		return err
	}

	for _, blk := range fnode.AllDataBlocks {
		volMap.SetAlloc(blk, false)
	}

	for _, blk := range fnode.AllIndirectBlocks {
		volMap.SetAlloc(blk, false)
	}

	err = volMap.Update()
	if err != nil {
		return err
	}

	for i := 0; i < NumPointers; i++ {
		fnode.Pointers[i].NumBlocks = 0
	}

	fnode.TotalSize = 0
	fnode.ThisSize = 0
	fnode.TotalBlocks = 0

	err = fnode.Update()
	if err != nil {
		return err
	}

	return nil
}
