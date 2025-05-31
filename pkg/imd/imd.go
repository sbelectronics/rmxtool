package imd

import (
	"fmt"
	"os"
)

type Sector struct {
	Deleted    bool
	SizeCode   uint8
	Data       []byte
	Bad        bool
	Number     uint8
	Compressed bool
}

type Track struct {
	Mode            uint8
	Cylinder        uint8
	Head            uint8
	SectorCount     uint8
	SectorSizeCode  uint8
	SectorNumbers   []uint8
	SectorSizeCodes []uint8
	Sectors         map[int]Sector
}

type ImageDisk struct {
	FileName  string
	Tracks    map[int]map[int]*Track
	CylCount  int
	HeadCount int
	Comment   []byte
}

func NewImageDisk() *ImageDisk {
	imd := &ImageDisk{}
	return imd
}

func (imd *ImageDisk) SetTrack(track *Track) {
	if imd.Tracks == nil {
		imd.Tracks = make(map[int]map[int]*Track)
	}
	cTrack, okay := imd.Tracks[int(track.Cylinder)]
	if !okay {
		cTrack = make(map[int]*Track)
		imd.Tracks[int(track.Cylinder)] = cTrack
	}
	cTrack[int(track.Head)] = track

	imd.CylCount = max(imd.CylCount, int(track.Cylinder+1))
	imd.HeadCount = max(imd.HeadCount, int(track.Head+1))
}

func (imd *ImageDisk) Load(fileName string) error {
	imd.FileName = fileName
	file, err := os.Open(imd.FileName)
	if err != nil {
		return fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	data, err := os.ReadFile(imd.FileName)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) < 5 {
		return fmt.Errorf("file too short: expected at least 5 bytes, got %d", len(data))
	}
	if string(data[:4]) != "IMD " {
		return fmt.Errorf("invalid file format: expected 'IMD ', got '%s'", data[:4])
	}
	data = data[4:]
	for len(data) > 0 && data[0] != 0x1A {
		imd.Comment = append(imd.Comment, data[0])
		data = data[1:]
	}
	imd.Comment = append(imd.Comment, 0x1A)
	data = data[1:] // Skip the 0x1A byte

	for len(data) > 0 {
		track := &Track{
			Mode:           data[0],
			Cylinder:       data[1],
			Head:           data[2],
			SectorCount:    data[3],
			SectorSizeCode: data[4],
			Sectors:        make(map[int]Sector),
		}
		imd.SetTrack(track)
		data = data[5:]

		fmt.Printf("Loading track: Mode=%d, Cylinder=%d, Head=%d, SectorCount=%d, SectorSizeCode=%d\n",
			track.Mode, track.Cylinder, track.Head, track.SectorCount, track.SectorSizeCode)

		if track.SectorSizeCode == 0xFF {
			for i := 0; i < int(track.SectorCount); i++ {
				track.SectorSizeCodes = append(track.SectorSizeCodes, data[0])
				data = data[1:]
			}
		} else {
			for i := 0; i < int(track.SectorCount); i++ {
				track.SectorSizeCodes = append(track.SectorSizeCodes, track.SectorSizeCode)
			}
		}

		track.SectorNumbers = data[:track.SectorCount]
		data = data[track.SectorCount:]

		for i := 0; i < int(track.SectorCount); i++ {
			dataType := data[0]
			data = data[1:]

			fmt.Printf("Sector %d: DataType=%d\n", track.SectorNumbers[i], dataType)

			if dataType > 0x08 {
				return fmt.Errorf("invalid data type: %d", dataType)
			}
			bad := (dataType == 0x00 || dataType == 0x05 || dataType == 0x06 || dataType == 0x07 || dataType == 0x08)
			deleted := (dataType == 0x03 || dataType == 0x04 || dataType == 0x07 || dataType == 0x08)
			compressed := (dataType == 0x02 || dataType == 0x04 || dataType == 0x06 || dataType == 0x08)
			secSize := 128 << track.SectorSizeCodes[i]

			fmt.Printf("SecSize=%d, Deleted=%t, Bad=%t, Compressed=%t\n",
				secSize, deleted, bad, compressed)

			var secData []byte
			if compressed {
				secData = make([]byte, secSize)
				for j := 0; j < secSize; j++ {
					secData[j] = data[0]
				}
				data = data[1:]
			} else {
				secData = data[:secSize]
				data = data[secSize:]
			}

			sector := Sector{
				Deleted:    deleted,
				Bad:        bad,
				Data:       secData,
				Number:     track.SectorNumbers[i],
				Compressed: compressed,
			}

			track.Sectors[int(sector.Number)] = sector
		}
	}

	return nil
}

func (imd *ImageDisk) GetIMD() ([]byte, error) {
	data := []byte{}
	data = append(data, []byte("IMD ")...)
	data = append(data, imd.Comment...)
	//data = append(data, 0x1A)
	for i := 0; i < imd.CylCount; i++ {
		for j := 0; j < imd.HeadCount; j++ {
			fmt.Printf("Writing track: Cylinder=%d, Head=%d\n", i, j)
			track := imd.Tracks[i][j]
			data = append(data, track.Mode)
			data = append(data, track.Cylinder)
			data = append(data, track.Head)
			data = append(data, track.SectorCount)
			data = append(data, track.SectorSizeCode)

			if track.SectorSizeCode == 0xFF {
				for _, sizeCode := range track.SectorSizeCodes {
					data = append(data, sizeCode)
				}
			}

			for i := 0; i < int(track.SectorCount); i++ {
				data = append(data, track.SectorNumbers[i])
			}

			for k := 0; k < int(track.SectorCount); k++ {
				sector := track.Sectors[int(track.SectorNumbers[k])]

				dataType := 0
				if sector.Deleted {
					dataType = 0x03
				} else {
					dataType = 0x01
				}
				if sector.Bad {
					dataType |= 0x04
				}

				compress := true
				for _, b := range sector.Data {
					if b != sector.Data[0] {
						compress = false
						break
					}
				}

				if compress {
					dataType += 1 // this is suspicious
				}

				data = append(data, uint8(dataType))

				fmt.Printf("  %d: SectorNumber=%d, SizeCode=%d, DataType=%d, compress=%v len=%d\n",
					k, track.SectorNumbers[k], track.SectorSizeCodes[k], dataType, compress, len(sector.Data))

				if compress {
					data = append(data, sector.Data[0]) // Compressed data is just the first byte repeated
				} else {
					data = append(data, sector.Data...)
				}
			}
		}
	}

	return data, nil
}

func (imd *ImageDisk) GetData() []byte {
	data := []byte{}
	for i := 0; i < imd.CylCount; i++ {
		for j := 0; j < imd.HeadCount; j++ {
			track := imd.Tracks[i][j]
			for k := 0; k < int(track.SectorCount); k++ {
				data = append(data, track.Sectors[k+1].Data...)
			}
		}
	}
	return data
}

func (imd *ImageDisk) SetData(data []byte) {
	for i := 0; i < imd.CylCount; i++ {
		for j := 0; j < imd.HeadCount; j++ {
			track := imd.Tracks[i][j]
			for k := 0; k < int(track.SectorCount); k++ {
				secLen := len(track.Sectors[k+1].Data)
				copy(track.Sectors[k+1].Data, data[:secLen])
				data = data[secLen:]
			}
		}
	}
}
