/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type RnaMatch struct {
	rna  string
	low  int
	high int
}

type Genome struct {
	file string
	rna  string
}

func NewGenome() *Genome {
	return &Genome{}
}

func NewGenomeFromFile(file string) *Genome {
	genome := NewGenome()
	genome.file = file
	genome.appendRnaFromFile(file)
	return genome
}

func (genome *Genome) appendRnaFromFile(file string) {
	genome.rna += extractRna(readFileContent(file))
}

func readFileContent(file string) string {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
}

func extractRna(input string) string {
	scanner := bufio.NewScanner(strings.NewReader(input))
	var rna string
	var readRna bool
	for scanner.Scan() {
		text := scanner.Text()
		if text == "ORIGIN" {
			readRna = true
			continue
		}
		if text == "//" {
			readRna = false
		}
		if readRna {
			pieces := strings.Split(strings.Trim(text, " "), " ")
			rnaPieces := pieces[1:len(pieces)]
			rna += strings.Join(rnaPieces, "")
		}
	}
	return rna
}

func (genome *Genome) compareTo(compareGenome *Genome) string {
	var result string
	var totalMatches int
	var longestMatch int
	var matchIndex int
	matches := []RnaMatch{}
	var i int
	var j int
	min := 8
	for ; i < len(genome.rna); i += j {
		j = int(math.Min(float64(min), float64(len(genome.rna)-i)))
		var rnaSlice string
		var rnaIndex int
		for ; (i + j) <= len(genome.rna); j++ {
			rnaSlice = genome.rna[i:(i + j)]
			rnaIndex = strings.Index(compareGenome.rna, rnaSlice)
			if rnaIndex == -1 {
				break
			}
			matchIndex = rnaIndex
		}
		if j > min {
			longestMatch = int(math.Max(float64(j), float64(longestMatch)))
			match := RnaMatch{rnaSlice, matchIndex, j}
			matches = append(matches, match)
			result += strings.ToUpper(rnaSlice)
			totalMatches += j
		} else {
			result += rnaSlice
		}
	}

	sequentialMatches := matches[0:1]
	for i = 0; i < (len(matches) - 1); {
		i++
		next := i
		nextLowest := matches[next].low
		for j, match := range matches[next:] {
			if match.low < nextLowest {
				next = i + j
				nextLowest = match.low
			}
		}
		sequentialMatches = append(sequentialMatches, matches[next])
		i = next
	}

	fmt.Printf("Genome: %s (size: %d)\n", filepath.Base(genome.file), len(genome.rna))
	fmt.Printf("Compare to: %s (size: %d)\n", filepath.Base(compareGenome.file), len(compareGenome.rna))
	fmt.Printf("Total matches: %d\n", len(matches))
	fmt.Printf("Total Sequential matches: %d\n", len(sequentialMatches))
	// fmt.Printf("Match sequence: %s\n", fmt.Sprint(matchSequence))
	fmt.Printf("Longest match: %d\n", longestMatch)
	fmt.Printf("Match: %.2f%% (check size: %d)\n\n", (float64(totalMatches) / float64(len(result)) * float64(100)), len(result))

	// fmt.Printf("Result size: %d Check size: %d Compare size: %d\n", len(result), len(genome.rna), len(compareGenome.rna))

	// strconv.FormatFloat(output.duration, 'f', -1, 64)
	return result
	// img := image.NewRGBA(image.Rect(0, 0, 320, 240))
	// x, y := 100, 100
	// addLabel(img, x, y, "Test123")
	// png.Encode(os.Stdout, img)
}

// compareCmd represents the compare command
var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("compare called")

		genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
		// genome1 := NewGenomeFromFile("./genome/examples/HIV-1-AF033819.txt")
		// genome1 := NewGenomeFromFile("./genome/examples/Pangolin-CoV-MT072864.txt")

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/H1N1/H1N1-seg1-NC_026438.txt")
			genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg2-NC_026435.txt")
			genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg3-NC_026437.txt")
			genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg4-NC_026433.txt")
			genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg5-NC_026436.txt")
			genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg6-NC_026434.txt")
			genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg7-NC_026431.txt")
			genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg8-NC_026432.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HIV-1-AF033819.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/EBOLA-NC_002549.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HEP-C-NC_004102.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/Maesles-NC_001498.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/Rabies-NC_001542.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/SARS-CoV1-NC_004718.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/SARS-CoV1-AY278741.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/MERS-CoV-KT029139.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HCoV-OC43-AY391777.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HCoV-229E-MF542265.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HCoV-NL63-MG772808.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HCoV-HKU1-AY597011.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/RaTG13-MN996532.txt")
			genome1.compareTo(genome2)
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/Pangolin-CoV-MT072864.txt")
			genome1.compareTo(genome2)
		}()

		test := "abcdef"
		fmt.Printf("TEST: %s, %d\n", test[5:6], len(test))

		// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
		// genome2 := NewGenomeFromFile("./genome/examples/RaTG13-MN996532.txt")
		// genome2 := NewGenomeFromFile("./genome/examples/HIV-1-AF033819.txt")
		// genome2 := NewGenomeFromFile("./genome/examples/EBOLA-NC_002549.txt")
		// genome2 := NewGenomeFromFile("./genome/examples/HEP-C-NC_004102.txt")
		// genome2 := NewGenomeFromFile("./genome/examples/Maesles-NC_001498.txt")
		// genome2 := NewGenomeFromFile("./genome/examples/Rabies-NC_001542.txt")
		// genome2 := NewGenomeFromFile("./genome/examples/SARS-NC_004718.txt")
		// genome2 := NewGenomeFromFile("./genome/examples/SARS-AY278741.txt")
		// genome2 := NewGenomeFromFile("./genome/examples/H1N1/H1N1-seg1-NC_026438.txt")
		// genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg2-NC_026435.txt")
		// genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg3-NC_026437.txt")
		// genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg4-NC_026433.txt")
		// genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg5-NC_026436.txt")
		// genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg6-NC_026434.txt")
		// genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg7-NC_026431.txt")
		// genome2.appendRnaFromFile("./genome/examples/H1N1/H1N1-seg8-NC_026432.txt")
		// fmt.Printf("File contents: %s\n\n", genome1.rna)
		// fmt.Printf("File contents: %s\n\n", genome2.rna)
		// genome1.compareTo(genome2)
	},
}

func init() {
	rootCmd.AddCommand(compareCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// compareCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// compareCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
