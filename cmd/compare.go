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
	"regexp"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/spf13/cobra"
	"golang.org/x/image/draw"
	"golang.org/x/image/font/gofont/gobold"
)

var darkColors = []string{
	"#b71c1c",
	"#880e4f",
	"#4a148c",
	"#311b92",
	"#1a237e",
	"#0d47a1",
	"#01579b",
	"#006064",
	"#004d40",
	"#1b5e20",
	"#33691e",
	"#827717",
	"#f57f17",
	"#ff6f00",
	"#e65100",
	"#bf360c",
}

var normalColors = []string{
	"#f44336",
	"#e91e63",
	"#9c27b0",
	"#673ab7",
	"#3f51b5",
	"#2196f3",
	"#03a9f4",
	"#00bcd4",
	"#009688",
	"#4caf50",
	"#8bc34a",
	"#cddc39",
	"#ffeb3b",
	"#ffc107",
	"#ff9800",
	"#ff5722",
}

type HexColor struct {
	hex string
}

func NewHexColor(hex string) *HexColor {
	return &HexColor{hex: hex}
}

func (hexColor *HexColor) toColor() *RGBA {
	var r, g, b int64
	if len(hexColor.hex) == 4 {
		r, _ = strconv.ParseInt(string(hexColor.hex[1:2]), 16, 0)
		g, _ = strconv.ParseInt(string(hexColor.hex[2:3]), 16, 0)
		b, _ = strconv.ParseInt(string(hexColor.hex[3:4]), 16, 0)
	}
	if len(hexColor.hex) == 7 {
		r, _ = strconv.ParseInt(string(hexColor.hex[1:3]), 16, 0)
		g, _ = strconv.ParseInt(string(hexColor.hex[3:5]), 16, 0)
		b, _ = strconv.ParseInt(string(hexColor.hex[5:7]), 16, 0)
	}
	return &RGBA{
		rgba: &color.RGBA{uint8(r), uint8(g), uint8(b), 0xff},
	}
}

type RGBA struct {
	rgba *color.RGBA
}

func (c *RGBA) RGBA() (r, g, b, a uint32) {
	return c.rgba.RGBA()
}

type RnaMatch struct {
	rna         string
	sourceIndex int
	targetIndex int
	length      int
	distance    int
}

type Genome struct {
	file        string
	baseName    string
	rna         string
	rnaMatch    string
	matchMask   string
	segmentMask string
	compare     *Genome
	matches     []RnaMatch
	segments    []GenomeSegment
}

type GenomeSegment struct {
	key   string
	start int
	end   int
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

func readFileContent(file string) string {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
}

func (genome *Genome) appendRnaFromFile(file string) {
	genome.segments = []GenomeSegment{}

	input := readFileContent(file)
	scanner := bufio.NewScanner(strings.NewReader(input))
	// r1, _ := regexp.Compile("^\\s*CDS\\s+(.*)$")
	// r2, _ := regexp.Compile("^\\s*5'UTR\\s+(.*)$")
	r, _ := regexp.Compile("(.*)\\s+([0-9]+)\\.\\.([0-9]+)$")
	// cdsr2, _ := regexp.Compile("CDS\\s+join\\(( ([^\\),]+) \\.\\.([0-9]+)),?+\\)$")

	var rna string
	var rnaLength = int64(len(genome.rna))

	for scanner.Scan() {
		text := strings.Trim(scanner.Text(), " ")
		if text == "ORIGIN" {
			break
		}
		matches := r.FindStringSubmatch(text)
		if len(matches) != 4 {
			continue
		}
		name := strings.Trim(matches[1], " ")
		if name != "gene" {
			continue
		}
		start, _ := strconv.ParseInt(matches[2], 10, 0)
		start += rnaLength
		end, _ := strconv.ParseInt(matches[3], 10, 0)
		end += rnaLength
		genome.segments = append(genome.segments, GenomeSegment{name, int(start), int(end)})
		// genome.segments[int(start)] = GenomeSegment{name, int(start), int(end)}
		// fmt.Printf("%s %s %d %d\n", matches[0], name, start, end)
	}

	for scanner.Scan() {
		text := strings.Trim(scanner.Text(), " ")
		if text == "//" {
			break
		}
		pieces := strings.Split(text, " ")
		rnaPieces := pieces[1:len(pieces)]
		rna += strings.Join(rnaPieces, "")
	}

	genome.rna += rna
}

func (genome *Genome) findLongestFuzzyMatchAtPos(sourceIndex int, compareGenome *Genome, min int, tolerance int) RnaMatch {
	var match RnaMatch
	distance := 0
	source := genome.rna[sourceIndex:]
	target := compareGenome.rna
	length := int(math.Min(float64(min), float64(len(source))))
	for ; (length) <= len(source); length++ {
		needle := source[0:length]
		if targetIndex := strings.Index(target, needle); targetIndex >= 0 {
			match = RnaMatch{needle, sourceIndex, targetIndex, length, distance}
			continue
		} else if match.length > 0 {
			distance++
			if distance > tolerance {
				break
			}
			mismatchAt := match.targetIndex + length
			if mismatchAt+1 < len(target) && length+1 < len(source) {
				target = target[:mismatchAt] + source[length:length+1] + target[mismatchAt+1:]
				continue
			}
		}
		break
	}
	return match
}

func (genome *Genome) findLongestMatchAtPos(sourceIndex int, compareGenome *Genome, min int) RnaMatch {
	var match RnaMatch
	source := genome.rna[sourceIndex:]
	length := int(math.Min(float64(min), float64(len(source))))
	for ; (length) <= len(source); length++ {
		needle := source[0:length]
		if targetIndex := strings.Index(compareGenome.rna, needle); targetIndex >= 0 {
			match = RnaMatch{needle, sourceIndex, targetIndex, length, 0}
			continue
		}
		break
	}
	return match
}

func (genome *Genome) createSegmentMask() {
	var count int
	genome.segmentMask = strings.Repeat("0", len(genome.rna))
	for _, segment := range genome.segments {
		char := strconv.FormatInt(int64(count+1), 36)
		mask := strings.Repeat(char, segment.end-segment.start)
		genome.segmentMask = genome.segmentMask[:segment.start] + mask + genome.segmentMask[segment.end:]
		count++
	}
}

func (genome *Genome) findMatches(compareGenome *Genome) {
	genome.matchMask = ""
	genome.matches = []RnaMatch{}
	min := 8
	for i := 0; i < len(genome.rna); {
		// match := genome.findLongestFuzzyMatchAtPos(i, compareGenome, min, 20)
		match := genome.findLongestMatchAtPos(i, compareGenome, min)
		if match.length > min {
			genome.matches = append(genome.matches, match)
			genome.matchMask += strings.Repeat("1", match.length)
			i += match.length
			continue
		}
		genome.matchMask += "0"
		i++
	}
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

func addLabel(img *image.RGBA, text string, x float64, y float64) {
	dc := gg.NewContextForRGBA(img)
	font, _ := truetype.Parse(gobold.TTF)
	// font, _ := truetype.Parse(goregular.TTF)
	face := truetype.NewFace(font, &truetype.Options{
		Size: 28,
	})
	dc.SetFontFace(face)
	dc.SetRGBA255(255, 255, 255, 255)
	dc.DrawString(text, x, y)
}

func getColors() []RGBA {
	rgbColors := make([]RGBA, 16)
	for i, hex := range darkColors {
		hexColor := NewHexColor(hex)
		rgbColor := hexColor.toColor()
		rgbColors[i] = *rgbColor
		// rgbColors = append(rgbColors, *rgbColor)
	}
	return rgbColors
}

func getMatchColors() []RGBA {
	rgbColors := make([]RGBA, 0, 16)
	for _, hex := range normalColors {
		hexColor := NewHexColor(hex)
		rgbColor := hexColor.toColor()
		rgbColors = append(rgbColors, *rgbColor)
	}
	return rgbColors
}

func (c *RGBA) luminance(l float64) *RGBA {
	return &RGBA{
		rgba: &color.RGBA{
			uint8(math.Min(float64(c.rgba.R)*l, 255)),
			uint8(math.Min(float64(c.rgba.G)*l, 255)),
			uint8(math.Min(float64(c.rgba.B)*l, 255)),
			0xff,
		},
	}
}

func (genome *Genome) writeImage() {
	segmentColors := getColors()
	segmentMatchColors := getMatchColors()

	fmt.Print(segmentColors[0].rgba)
	fmt.Print("\n")
	fmt.Print(segmentMatchColors[0].rgba)
	fmt.Print("\n")
	fmt.Print(darkColors[0])
	fmt.Print("\n")
	fmt.Print(normalColors[0])
	fmt.Print("\n")

	segmentCount, _ := strconv.ParseInt(string(genome.segmentMask[1]), 36, 0)
	segmentCount = segmentCount % 16

	fmt.Print(segmentCount)
	fmt.Print("\n")

	width := int(math.Ceil(math.Sqrt(float64(len(genome.rna)))))
	height := width
	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})
	dc := gg.NewContextForRGBA(img)
	dc.Clear()
	dc.SetRGB(0, 0, 0)

	colors := map[string]*RGBA{
		"a": segmentColors[0].luminance(0.3),
		"t": segmentColors[0].luminance(0.4),
		"g": segmentColors[0].luminance(0.5),
		"c": segmentColors[0].luminance(0.6),
	}

	matchColors := map[string]*RGBA{
		"a": segmentMatchColors[0].luminance(0.7),
		"t": segmentMatchColors[0].luminance(0.8),
		"g": segmentMatchColors[0].luminance(0.9),
		"c": segmentMatchColors[0].luminance(1),
	}

	x := 0
	y := 0

	for i, char := range genome.rna {
		strChar := string(char)
		if i < len(genome.segmentMask) {
			segmentCount, _ := strconv.ParseInt(string(genome.segmentMask[i]), 36, 0)
			segmentCount = segmentCount % 16
			// fmt.Printf("%d", segmentCount)
			// r, g, b, _ := segmentColors[segmentCount].RGBA()
			segmentColor := segmentColors[segmentCount]
			segmentMatchColor := segmentMatchColors[segmentCount]
			// r, g, b, _ := palette.WebSafe[(segmentCount*12)-1].RGBA()
			colors = map[string]*RGBA{
				"a": segmentColor.luminance(0.3),
				"t": segmentColor.luminance(0.4),
				"g": segmentColor.luminance(0.5),
				"c": segmentColor.luminance(0.6),
			}
			matchColors = map[string]*RGBA{
				"a": segmentMatchColor.luminance(0.7),
				"t": segmentMatchColor.luminance(0.8),
				"g": segmentMatchColor.luminance(0.9),
				"c": segmentMatchColor.luminance(1),
			}
		}

		if !strings.Contains("atgc", strChar) {
			img.Set(x, y, color.RGBA{0, 0, 0, 0xff})
		} else {
			matchMaskChar := "0"
			if i < len(genome.matchMask) {
				matchMaskChar = string(genome.matchMask[i])
			}
			switch matchMaskChar {
			case "1":
				img.Set(x, y, matchColors[strChar])
				break
			default:
				img.Set(x, y, colors[strChar])
				break
			}
		}

		x++
		if x >= width {
			y++
			x = 0
		}
	}

	sb := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, sb.Dx()*5, sb.Dy()*5))
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, sb, draw.Over, nil)

	addLabel(dst, fmt.Sprintf("%s (%d)", genome.baseName, len(genome.rna)), 10, 30)

	baseName := genome.baseName
	if genome.compare != nil && genome.compare.baseName != "" {
		baseName = genome.baseName + "-" + genome.compare.baseName
		addLabel(dst, fmt.Sprintf("%s (%d)", genome.compare.baseName, len(genome.compare.rna)), 10, 60)
	}

	totalMatchSize := genome.getTotalMatchSize()
	addLabel(dst, fmt.Sprintf("Longest match: %d", genome.getLongestMatch()), 10, 90)
	addLabel(dst, fmt.Sprintf("Match: %d (%.2f%%)", totalMatchSize, (float64(totalMatchSize)/float64(len(genome.rna))*float64(100))), 10, 120)

	filePath := "./genome/images/" + baseName + ".png"
	fmt.Printf("Writing image: %s\n", filePath)
	f, _ := os.Create(filePath)
	png.Encode(f, dst)
}

func (genome *Genome) compareTo(compareGenome *Genome) string {

	genome.compare = compareGenome
	genome.findMatches(compareGenome)
	longestMatch := genome.getLongestMatch()
	totalMatchSize := genome.getTotalMatchSize()

	// fmt.Printf("%s\n\n", result)
	// fmt.Printf("%s\n\n", genome.segmentMask)

	fmt.Printf("\nGenome: %s (size: %d)\n", genome.baseName, len(genome.rna))
	fmt.Printf("Compare to: %s (size: %d)\n", compareGenome.baseName, len(compareGenome.rna))
	fmt.Printf("Total matches: %d\n", len(genome.matches))
	// fmt.Printf("Match sequence: %s\n", fmt.Sprint(matchSequence))
	fmt.Printf("Longest match: %d\n", longestMatch)
	fmt.Printf("Match: %.2f%%\n", (float64(totalMatchSize) / float64(len(genome.rna)) * float64(100)))
	fmt.Printf("Check RNA size: %d\n", len(genome.rna))
	fmt.Printf("Check match size: %d\n", len(genome.matchMask))
	fmt.Printf("Check segment size: %d\n", len(genome.segmentMask))
	fmt.Printf("%s\n", genome.rna[:100])
	fmt.Printf("%s\n", genome.matchMask[:100])
	fmt.Printf("%s\n", genome.segmentMask[:100])

	// genome.writeCompareImage(compareGenome)
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

		// test := "abcdef"
		// fmt.Printf("TEST: %d, %s, %s\n", len(test), test[:7], test[7:])
		// // fmt.Printf("TEST: %s, %d\n", test[5:6], len(test))
		// os.Exit(0)

		genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
		genome1.createSegmentMask()

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
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		// os.Exit(0)

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HIV-1-AF033819.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/EBOLA-NC_002549.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HEP-C-NC_004102.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/Maesles-NC_001498.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/Rabies-NC_001542.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/SARS-CoV1-NC_004718.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/SARS-CoV1-AY278741.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/MERS-CoV-KT029139.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HCoV-OC43-AY391777.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HCoV-229E-MF542265.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HCoV-NL63-MG772808.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/HCoV-HKU1-AY597011.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/RaTG13-MN996532.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/Pangolin-CoV-MT072864.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1)
			genome2.writeImage()
		}()

		func() {
			// genome1 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome2 := NewGenomeFromFile("./genome/examples/SARS-CoV2-MN908947.txt")
			genome1.compareTo(genome2)
			genome1.writeImage()
			// genome2.createSegmentMask()
			// genome2.compareTo(genome1)
			// genome2.writeImage()
		}()

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
