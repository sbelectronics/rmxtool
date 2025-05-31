# rmxtool
### Scott Baker, https://www.smbaker.com/

This is a tool for manipulating disk images for the iRMX-86 operating system.

## Prebuilt Binaries

Prebuilt binaries are in the release directory:

* Windows: release/windows/amd64/rmxtool.exe

* Linux, x86: release/linux/amd64/rmxtool

* Linux, ARM: release/linux/arm64/rmxtool

## Short Demo

This demo starts with a file, `test.img`, that is a raw image from disk 147023, which is
the iRMX x86 installer disk.

```bash
$ echo "Hello, World" > hello.txt

$ rmxtool put hello.txt
Stored 13 bytes to FNode 7 (hello.txt)

$ rmxtool dir
Name               FNode     Size  Type        Flags  Accessors
----               -----     ----  ----        -----  ---------
R?SPACEMAP             1       80  VolMap      A P    R:65535
R?FNODEMAP             2        6  FNodeMap    A P    R:65535
R?BADBLOCKMAP          4       80  BadBlock    A P    R:65535
R?VOLUMELABEL          5     3328  VolLabel    A P    R:65535
hello.txt              7       13  Data        A P    DRAU:0 DRAU:65535

$ rmxtool get hello.txt -o -
Hello, World
Wrote 13 bytes to stdout
```
