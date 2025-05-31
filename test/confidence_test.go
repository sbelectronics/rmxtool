package confidence

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/stretchr/testify/suite"
	"os"
	"os/exec"
	"path"
	"testing"
)

const (
	RMXTOOL   = "../build/_output/rmxtool"
	SRCIMAGE  = "../test.save"
	TESTIMAGE = "../test.work"
)

var (
	SRCIMAGE_FILES = map[string]string{
		"/system/diskverify":         "0352e11292c716a69f15aba12ca7b802ddfcd252", // v - lines marked with V verified with Mark's archive
		"/system/rmx86":              "fb4606acf243c372a93a66c641eae956ebc7ba21",
		"/system/attachdevice":       "39514796a24bd8f831d8a93e915f5aeb641835a7", // v
		"/system/copy":               "9edc18818957df1f2da712f96293a3fd0a0de705", // v
		"/system/date":               "e227be390666e0e04f4dd5cb7261ec4f9b6d4954", // v
		"/system/detachdevice":       "a810719cbd1bbe94e583f789f358e84b6399ca01", // v
		"/system/format":             "dde5948541640a29844cfb8f1d100413d77b6095", // v
		"/system/permit":             "0db1ec8f6da74b3fc0ab00751ab4e3876238b13a", // v
		"/system/submit":             "a032af1b27d02dd070ff94a9faa439a5ce544b3b", // v
		"/system/super":              "d718586713449dded8d045cd7ecfee22b0b881ef", // v
		"/system/time":               "e6c89eebb518d86c14a3fe9ddfb0feb4eb5c0ffa", // v
		"/user/world/prog/R?LOGON":   "705734b1cd5728f4834d891391f40adeee70f0ce",
		"/config/user/0":             "626d0255a8adf3e4e97f5b5e69455efec8cf88bf",
		"/config/user/world":         "60dc2796162848a811889d47ea129f61d6175b2c",
		"/config/cmd/instal860.csd":  "c02da3d4173bdf3ee61a5d86c57c06c386794f79",
		"/config/cmd/instal860u.csd": "c7983c30190ec35e9f21fe04441c1ea095aac987",
		"/config/cmd/instal863.csd":  "ebc01f3cd60d7f893c84e1ec57f7dd9c16f3f772",
		"/config/TERMINALS":          "d8e009efc21678650bc066de87b1d938b2250f77",
		"/instal.csd":                "cb4e145bc36ec7fe614239dcf9158cd4f385df68",
	}
)

type ConfidenceSuite struct {
	suite.Suite
}

func (s *ConfidenceSuite) SetupTest() {
	err := os.Remove(TESTIMAGE)
	if err != nil && !os.IsNotExist(err) {
		s.FailNow("Failed to remove TESTIMAGE", err)
	}

	input, err := os.ReadFile(SRCIMAGE)
	s.Require().NoError(err, "Failed to read SRCIMAGE")

	err = os.WriteFile(TESTIMAGE, input, 0644)
	s.Require().NoError(err, "Failed to write TESTIMAGE")
}

func (s *ConfidenceSuite) run(args ...string) (string, string, error) {
	cmd := exec.Command(RMXTOOL, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func (s *ConfidenceSuite) VerifyFiles(files map[string]string) {
	tempDir, err := os.MkdirTemp("", "confidence-test")
	s.Require().NoError(err)
	defer func() {
		err := os.RemoveAll(tempDir)
		s.NoError(err, "Failed to clean up temporary directory")
	}()

	for fileName, expectedHash := range files {
		baseName := path.Base(fileName)
		destName := path.Join(tempDir, baseName)

		out, errOut, err := s.run("get", "-q", fileName, "-f", TESTIMAGE, "-o", destName)

		if expectedHash == "" {
			s.Error(err, "Expected error for file: %s", fileName)
		} else {
			s.NoError(err)
			s.Empty(errOut)
			s.ShowIfError(err, out, errOut)

			if err == nil {
				_, err = os.Stat(destName)
				s.NoError(err, "File should be created: %s", destName)

				content, err := os.ReadFile(destName)
				s.NoError(err, "Failed to read output file: %s", destName)

				// Verify the file content matches the expected hash
				hash := fmt.Sprintf("%x", sha1.Sum(content))
				s.Equal(expectedHash, hash, "File hash mismatch for: %s", destName)
			}
		}
	}
}

func (s *ConfidenceSuite) ShowIfError(err error, out, errOut string) {
	if err != nil {
		s.T().Logf("\nOutput: %s\n", out)
		s.T().Logf("Error Output: %s\n", errOut)
	}
}

func (s *ConfidenceSuite) CheckDisk() {
	out, errOut, err := s.run("chkdsk", "-f", TESTIMAGE)
	s.NoError(err, "Chkdsk command failed")
	s.ShowIfError(err, out, errOut)

	s.Contains(out, "successfully", "Check command output should indicate disk is OK")
}

func (s *ConfidenceSuite) TestGet() {
	s.VerifyFiles(SRCIMAGE_FILES)
}

func (s *ConfidenceSuite) TestDelete() {
	s.VerifyFiles(SRCIMAGE_FILES)

	files := make(map[string]string)
	for k, v := range SRCIMAGE_FILES {
		files[k] = v
	}

	out, errOut, err := s.run("delete", "-q", "/system/diskverify", "-f", TESTIMAGE)
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	files["/system/diskverify"] = ""
	s.VerifyFiles(files)

	out, errOut, err = s.run("delete", "-q", "/config/user/world", "-f", TESTIMAGE)
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	files["/config/user/world"] = ""
	s.VerifyFiles(files)
	s.CheckDisk()
}

func (s *ConfidenceSuite) TestPut() {
	files := make(map[string]string)
	for k, v := range SRCIMAGE_FILES {
		files[k] = v
	}

	files["country.txt"] = "1219be1aa7e85838ad7e5940ca078e1259cedf23"

	out, errOut, err := s.run("put", "-q", "testdata/country.txt", "-f", TESTIMAGE)
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	s.VerifyFiles(files)
	s.CheckDisk()
}

func (s *ConfidenceSuite) TestDeletePut() {
	files := make(map[string]string)
	for k, v := range SRCIMAGE_FILES {
		files[k] = v
	}

	out, errOut, err := s.run("delete", "-q", "/system/diskverify", "-f", TESTIMAGE)
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	files["country.txt"] = "1219be1aa7e85838ad7e5940ca078e1259cedf23"
	files["/system/lamb.txt"] = "6002f8f827625b854c2764e3baa3611bdc7728ab"
	files["/user/world/odyssey.txt"] = "230f4a98d3566dec50b3eb0e750df902cc652169"
	files["/lang/scott.txt"] = "aa630cac89431f84f6d20c12c837311e5e44bfd6"

	files["/system/diskverify"] = ""

	out, errOut, err = s.run("put", "-q", "testdata/country.txt", "-f", TESTIMAGE)
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	out, errOut, err = s.run("put", "-q", "testdata/lamb.txt", "-f", TESTIMAGE, "-d", "/system")
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	out, errOut, err = s.run("put", "-q", "testdata/odyssey.txt", "-f", TESTIMAGE, "-d", "/user/world")
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	out, errOut, err = s.run("put", "-q", "testdata/scott.txt", "-f", TESTIMAGE, "-d", "/lang")
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	s.VerifyFiles(files)
	s.CheckDisk()
}

func (s *ConfidenceSuite) TestWipeAndFill() {
	out, errOut, err := s.run("wipe", "-f", TESTIMAGE)
	s.NoError(err, "Wipe command failed")
	s.ShowIfError(err, out, errOut)

	// Verify that the image is empty
	files := make(map[string]string)
	for k := range SRCIMAGE_FILES {
		files[k] = ""
	}

	files["country.txt"] = "1219be1aa7e85838ad7e5940ca078e1259cedf23"
	files["/system/lamb.txt"] = "6002f8f827625b854c2764e3baa3611bdc7728ab"
	files["/user/world/odyssey.txt"] = "230f4a98d3566dec50b3eb0e750df902cc652169"
	files["/lang/scott.txt"] = "aa630cac89431f84f6d20c12c837311e5e44bfd6"

	out, errOut, err = s.run("mkdir", "-q", "system", "-f", TESTIMAGE)
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	out, errOut, err = s.run("mkdir", "-q", "user", "-f", TESTIMAGE)
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	out, errOut, err = s.run("mkdir", "-q", "user/world", "-f", TESTIMAGE)
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	out, errOut, err = s.run("mkdir", "-q", "lang", "-f", TESTIMAGE)
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	out, errOut, err = s.run("put", "-q", "testdata/country.txt", "-f", TESTIMAGE)
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	out, errOut, err = s.run("put", "-q", "testdata/lamb.txt", "-f", TESTIMAGE, "-d", "/system")
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	out, errOut, err = s.run("put", "-q", "testdata/odyssey.txt", "-f", TESTIMAGE, "-d", "/user/world")
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	out, errOut, err = s.run("put", "-q", "testdata/scott.txt", "-f", TESTIMAGE, "-d", "/lang")
	s.NoError(err)
	s.ShowIfError(err, out, errOut)

	s.VerifyFiles(files)

	s.CheckDisk()
}

func TestConfidenceSuite(t *testing.T) {
	suite.Run(t, new(ConfidenceSuite))
}
