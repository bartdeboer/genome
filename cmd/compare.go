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
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/spf13/cobra"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/math/fixed"
)

type RnaMatch struct {
	rna    string
	index  int
	length int
}

type Genome struct {
	file     string
	baseName string
	rna      string
	rnaMatch string
	matches  map[int]RnaMatch
}

func NewGenome() *Genome {
	return &Genome{}
}

func NewGenomeFromFile(file string) *Genome {
	genome := NewGenome()
	genome.file = file
	genome.baseName = filepath.Base(strings.TrimSuffix(file, filepath.Ext(file)))
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

func (genome *Genome) findLongestMatchAtPos(pos int, compareGenome *Genome, min int) RnaMatch {
	var match RnaMatch
	// var matchLength int
	sourceHaystack := genome.rna[pos:]
	length := int(math.Min(float64(min), float64(len(sourceHaystack))))
	for ; (length) <= len(sourceHaystack); length++ {
		needle := sourceHaystack[0:length]
		if index := strings.Index(compareGenome.rna, needle); index >= 0 {
			match = RnaMatch{needle, pos, length}
			continue
		}
		break
	}
	return match
}

func (genome *Genome) findMatches(compareGenome *Genome) {
	var result string
	genome.matches = map[int]RnaMatch{}
	matches := []RnaMatch{}
	min := 8
	for i := 0; i < len(genome.rna); {
		match := genome.findLongestMatchAtPos(i, compareGenome, min)
		if match.length > min {
			genome.matches[i] = match
			matches = append(matches, match)
			result += strings.ToUpper(match.rna)
			i += match.length
			continue
		}
		result += genome.rna[i:(i + 1)]
		i++
	}
	genome.rnaMatch = result
}

func (genome *Genome) getTotalMatchSize() int {
	var size int
	for _, match := range genome.matches {
		size += match.length
	}
	return size
}

func (genome *Genome) getLongestMatch() int {
	var size int
	for _, match := range genome.matches {
		size = int(math.Max(float64(size), float64(match.length)))
	}
	return size
}

func (genome *Genome) findSequentialMatches(compareGenome *Genome) string {
	// sequentialMatches := matches[0:1]
	// for i = 0; i < (len(matches) - 1); {
	// 	i++
	// 	next := i
	// 	nextLowest := matches[next].low
	// 	for j, match := range matches[next:] {
	// 		if match.low < nextLowest {
	// 			next = i + j
	// 			nextLowest = match.low
	// 		}
	// 	}
	// 	sequentialMatches = append(sequentialMatches, matches[next])
	// 	i = next
	// }
	return ""
}

func addLabel2(img *image.RGBA, text string) {
	dc := gg.NewContextForRGBA(img)
	// const S = 1024
	// dc := gg.NewContext(S, S)
	// dc.SetRGB(1, 1, 1)
	// dc.Clear()

	// font, _ := truetype.Parse(goregular.TTF)
	font, _ := truetype.Parse(gobold.TTF)
	face := truetype.NewFace(font, &truetype.Options{
		Size: 40,
	})
	dc.SetFontFace(face)
	dc.SetRGBA255(255, 255, 255, 255)
	dc.DrawString(text, 10, 50)
}

func addLabel(img *image.RGBA, x, y int, label string) {
	col := color.RGBA{255, 255, 255, 255}
	point := fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}

func (genome *Genome) writeCompareImage(compareGenome *Genome) {
	width := int(math.Ceil(math.Sqrt(float64(len(genome.rna)))))
	height := width
	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})

	colors := map[string]color.RGBA{
		"a": color.RGBA{114, 27, 101, 0xff},
		"t": color.RGBA{248, 97, 90, 0xff},
		"g": color.RGBA{184, 13, 87, 0xff},
		"c": color.RGBA{255, 217, 104, 0xff},
		"A": color.RGBA{25, 70, 112, 0xff},
		"T": color.RGBA{76, 158, 234, 0xff},
		"G": color.RGBA{1, 90, 172, 0xff},
		"C": color.RGBA{189, 223, 255, 0xff},
	}

	x := 0
	y := 0

	for _, char := range genome.rnaMatch {
		img.Set(x, y, colors[string(char)])
		x++
		if x >= width {
			y++
			x = 0
		}
	}

	sb := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, sb.Dx()*5, sb.Dy()*5))
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, sb, draw.Over, nil)

	addLabel2(dst, compareGenome.baseName)

	// Encode as PNG.
	filePath := "./genome/images/" + genome.baseName + "-" + compareGenome.baseName + ".png"
	fmt.Printf("Writing image: %s\n", filePath)
	f, _ := os.Create(filePath)
	png.Encode(f, dst)
}

func (genome *Genome) compareTo(compareGenome *Genome) string {

	genome.findMatches(compareGenome)
	longestMatch := genome.getLongestMatch()
	totalMatchSize := genome.getTotalMatchSize()

	// fmt.Printf("%s\n\n", result)

	fmt.Printf("Genome: %s (size: %d)\n", genome.baseName, len(genome.rna))
	fmt.Printf("Compare to: %s (size: %d)\n", compareGenome.baseName, len(compareGenome.rna))
	fmt.Printf("Total matches: %d\n", len(genome.matches))
	// fmt.Printf("Match sequence: %s\n", fmt.Sprint(matchSequence))
	fmt.Printf("Longest match: %d\n", longestMatch)
	fmt.Printf("Match: %.2f%% (check size: %d)\n\n", (float64(totalMatchSize) / float64(len(genome.rnaMatch)) * float64(100)), len(genome.rnaMatch))

	genome.writeCompareImage(compareGenome)

	// fmt.Printf("Result size: %d Check size: %d Compare size: %d\n", len(result), len(genome.rna), len(compareGenome.rna))

	// strconv.FormatFloat(output.duration, 'f', -1, 64)
	return genome.rnaMatch
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

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
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
