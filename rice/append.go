package main

import (
	"archive/zip"
	"fmt"
	"go/build"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/daaku/go.zipexe"
)

func operationAppend(pkg *build.Package) {
	if runtime.GOOS == "windows" {
		_, err := exec.LookPath("zip")
		if err != nil {
			fmt.Println("#### WARNING ! ####")
			fmt.Println("`rice append` is known not to work under windows because the `zip` command is not available. Please let me know if you got this to work (and how).")
		}
	}

	// MARKED FOR DELETION
	// This is actually not required, the append command now has the option --exec required.
	// // check if package is a command
	// if !pkg.IsCommand() {
	// 	fmt.Println("Error: can not append to non-main package. Please follow instructions at github.com/GeertJohan/go.rice")
	// 	os.Exit(1)
	// }

	// find boxes for this command
	boxMap := findBoxes(pkg)

	// notify user when no calls to rice.FindBox are made (is this an error and therefore os.Exit(1) ?
	if len(boxMap) == 0 {
		fmt.Println("no calls to rice.FindBox() or rice.MustFindBox() found")
		return
	}

	verbosef("\n")

	// find abs path for binary file
	binfileName, err := filepath.Abs(flags.Append.Executable)
	if err != nil {
		fmt.Printf("Error finding absolute path for executable to append: %s\n", err)
		os.Exit(1)
	}
	verbosef("Will append to file: %s\n", binfileName)

	// check that command doesn't already have zip appended
	if rd, _ := zipexe.Open(binfileName); rd != nil {
		fmt.Printf("Cannot append to already appended executable. Please remove %s and build a fresh one.\n", binfileName)
		os.Exit(1)
	}

	// open binfile
	binfile, err := os.OpenFile(binfileName, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Printf("Error: unable to open executable file: %s\n", err)
		os.Exit(1)
	}

	// create zip.Writer
	binfileInfo, err := binfile.Stat()
	if err != nil {
		fmt.Printf("error stat binfile: %v", err)
		os.Exit(1)
	}
	zipWriter := zip.NewWriterWithOffset(binfile, binfileInfo.Size())

	for boxname := range boxMap {
		appendedBoxName := strings.Replace(boxname, `/`, `-`, -1)

		// walk box path's and insert files
		boxPath := filepath.Clean(filepath.Join(pkg.Dir, boxname))
		filepath.Walk(boxPath, func(path string, info os.FileInfo, err error) error {
			// create zipFilename
			zipFileName := filepath.Join(appendedBoxName, strings.TrimPrefix(path, boxPath))

			// write directories as empty file with comment "dir"
			if info.IsDir() {
				_, err := zipWriter.CreateHeader(&zip.FileHeader{
					Name:    zipFileName,
					Comment: "dir",
				})
				if err != nil {
					fmt.Printf("Error creating dir in tmp zip: %s\n", err)
					os.Exit(1)
				}
				return nil
			}

			// create zipFileWriter
			zipFileHeader, err := zip.FileInfoHeader(info)
			if err != nil {
				fmt.Printf("Error creating zip FileHeader: %v\n", err)
				os.Exit(1)
			}
			zipFileHeader.Name = zipFileName
			zipFileWriter, err := zipWriter.CreateHeader(zipFileHeader)
			if err != nil {
				fmt.Printf("Error creating file in tmp zip: %s\n", err)
				os.Exit(1)
			}
			srcFile, err := os.Open(path)
			if err != nil {
				fmt.Printf("Error opening file to append: %s\n", err)
				os.Exit(1)
			}
			_, err = io.Copy(zipFileWriter, srcFile)
			if err != nil {
				fmt.Printf("Error copying file contents to zip: %s\n", err)
				os.Exit(1)
			}
			srcFile.Close()

			return nil
		})
	}

	err = zipWriter.Close()
	if err != nil {
		fmt.Printf("Error closing zip writer: %s\n", err)
		os.Exit(1)
	}
}
