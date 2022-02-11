package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func ReadLine(reader io.Reader, f func(string)) {
	buf := bufio.NewReader(reader)
	line, err := buf.ReadBytes('\n')
	for err == nil {
		line = bytes.TrimRight(line, "\n")
		if len(line) > 0 {
			if line[len(line)-1] == 13 { //'\r'
				line = bytes.TrimRight(line, "\r")
			}
			f(string(line))
		}
		line, err = buf.ReadBytes('\n')
	}

	if len(line) > 0 {
		f(string(line))
	}
}

func Po2Csv(rootPath string) error {
	langs := make([][]string, 0)
	var firstLine = []string{"Key", "Source", "Source2", "msgctxt", "msgid"}
	csvContent := make([][]string, 0)
	if err := filepath.Walk(rootPath, func(path string, info fs.FileInfo, err error) error {
		_, lan := filepath.Split(path)
		if info.IsDir() && path != rootPath {
			firstLine = append(firstLine, lan)
			file := path + "/Game.po"
			lines := make([]string, 0)
			if content, _err := ioutil.ReadFile(file); _err != nil {
				log.Fatal("readfile " + file + _err.Error())
			} else {
				ReadLine(bytes.NewReader(content), func(line string) {
					lines = append(lines, line)
				})
			}
			langs = append(langs, lines)
		}
		return nil
	}); err != nil {
		return err
	}
	csvContent = append(csvContent, firstLine)
	var newLine []string
	for i := 0; i < len(langs[0]); i++ {
		line := langs[0][i]
		if strings.HasPrefix(line, "#. Key:\t") {
			newLine = make([]string, 0)
			newLine = append(newLine, line[len("#. Key: "):])
		}
		if strings.HasPrefix(line, "#. SourceLocation:\t") {
			newLine = append(newLine, line[len("#. SourceLocation:\t"):])
		}
		if strings.HasPrefix(line, "#: ") {
			newLine = append(newLine, line[len("#: "):])
		}
		if strings.HasPrefix(line, "msgctxt ") && len(newLine) > 0 {
			newLine = append(newLine, line[len("msgctxt ")+1:len(line)-1])
		}
		if strings.HasPrefix(line, "msgid ") && len(newLine) > 0 {
			newLine = append(newLine, line[len("msgid ")+1:len(line)-1])
		}
		if strings.HasPrefix(line, "msgstr ") && len(newLine) > 0 {
			newLine = append(newLine, line[len("msgstr ")+1:len(line)-1])
			for j := 1; j < len(langs); j++ {
				otherLine := langs[j][i]
				newLine = append(newLine, otherLine[len("msgstr ")+1:len(otherLine)-1])
			}
			csvContent = append(csvContent, newLine)
		}
	}
	fileName := rootPath + "/localization.csv"
	os.Remove(fileName)
	fileWrite, _err := os.OpenFile(fileName, os.O_CREATE, fs.ModePerm)
	if _err != nil {
		return _err
	}
	w := csv.NewWriter(fileWrite)
	if err := w.WriteAll(csvContent); err != nil {
		return err
	}
	return fileWrite.Close()
}

func Csv2Po(rootPath string) error {
	fileName := rootPath + "/localization.csv"

	fileReader, _err := os.OpenFile(fileName, os.O_RDONLY, fs.ModePerm)
	if _err != nil {
		return _err
	}
	r := csv.NewReader(fileReader)
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return err
	}
	if len(records[0]) < 1 {
		return fmt.Errorf("wrong csv %s", fileName)
	}
	firstline := records[0]
	langs := make([]*os.File, 0)
	for i := 5; i < len(firstline); i++ {
		file := rootPath + "/" + firstline[i] + "/Game.po"
		oldFile := file
		header := ""
		if fileRead, _er := ioutil.ReadFile(oldFile); _er == nil {
			content := string(fileRead)
			header = content[:strings.Index(content, "#. Key:")-1]
		}
		os.Remove(file)
		fileWrite, _e := os.OpenFile(file, os.O_CREATE, fs.ModePerm)
		if _e != nil {
			return _e
		}
		fileWrite.WriteString(header)
		langs = append(langs, fileWrite)

	}
	for i := 1; i < len(records); i++ {
		line := records[i]
		header := fmt.Sprintf(`
#. Key:	%s
#. SourceLocation:	%s
#: %s
msgctxt "%s"
msgid "%s"
`, line[0], line[1], line[2], line[3], line[4])
		for j := 0; j < len(langs); j++ {
			chunk := fmt.Sprintf("%smsgstr \"%s\"\n", header, line[5+j])
			langs[j].WriteString(chunk)
		}
	}
	for j := 0; j < len(langs); j++ {
		langs[j].WriteString("\n")
		langs[j].Close()
	}
	return nil
}
func main() {
	var rootPath = ""
	var po2csv = true
	if len(os.Args) == 3 {
		rootPath = os.Args[1]
		po2csv = os.Args[2] == "po2csv"
		if info, err := os.Stat(rootPath); err != nil {
			log.Fatalf("%s", err.Error())
		} else if !info.IsDir() {
			log.Fatalf("%s not a dir", rootPath)
		}
	} else {
		log.Fatal("Po2Csv path po2csv/csv2po")
	}
	if po2csv {
		if err := Po2Csv(rootPath); err != nil {
			log.Fatalf("Po2Csv failed %s", err.Error())
		}
		log.Printf("Po2Csv succeed to %s", rootPath+"/localization.csv")
	} else {
		if err := Csv2Po(rootPath); err != nil {
			log.Fatalf("Csv2Po failed %s", err.Error())
		}
		log.Printf("Csv2Po succed")
	}
}
