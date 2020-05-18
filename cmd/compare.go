// Copyright 2020 Bart de Boer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/spf13/cobra"
	"golang.org/x/image/draw"
	"golang.org/x/image/font/gofont/gobold"
)

// Amino Acid Codes
// https://www.ddbj.nig.ac.jp/ddbj/code-e.html

var codonTranslation = map[string]string{
	"TTT": "F",
	"TTC": "F",
	"TTA": "L",
	"TTG": "L",
	"TCT": "S",
	"TCC": "S",
	"TCA": "S",
	"TCG": "S",
	"TAT": "Y",
	"TAC": "Y",
	"TAA": "X",
	"TAG": "X",
	"TGT": "C",
	"TGC": "C",
	"TGA": "X",
	"TGG": "W",
	"CTT": "L",
	"CTC": "L",
	"CTA": "L",
	"CTG": "L",
	"CCT": "P",
	"CCC": "P",
	"CCA": "P",
	"CCG": "P",
	"CAT": "H",
	"CAC": "H",
	"CAA": "Q",
	"CAG": "Q",
	"CGT": "R",
	"CGC": "R",
	"CGA": "R",
	"CGG": "R",
	"ATT": "I",
	"ATC": "I",
	"ATA": "I",
	"ATG": "M",
	"ACT": "T",
	"ACC": "T",
	"ACA": "T",
	"ACG": "T",
	"AAT": "N",
	"AAC": "N",
	"AAA": "K",
	"AAG": "K",
	"AGT": "S",
	"AGC": "S",
	"AGA": "R",
	"AGG": "R",
	"GTT": "V",
	"GTC": "V",
	"GTA": "V",
	"GTG": "V",
	"GCT": "A",
	"GCC": "A",
	"GCA": "A",
	"GCG": "A",
	"GAT": "D",
	"GAC": "D",
	"GAA": "E",
	"GAG": "E",
	"GGT": "G",
	"GGC": "G",
	"GGA": "G",
	"GGG": "G",
	// "n/a": "B",
	// "n/a": "Z",
}

func getAminoAcidCharSet() []string {
	charSet := []string{}
	existing := make(map[string]bool)
	for _, code := range codonTranslation {
		if _, ok := existing[code]; ok {
			continue
		}
		charSet = append(charSet, code)
		existing[code] = true
	}
	return charSet
}

func getDnaCharSet() []string {
	return []string{"a", "t", "g", "c"}
}

var dnaMRnaMap = map[string]string{
	"a": "u",
	"t": "a",
	"g": "c",
	"c": "g",
}

var mRnaRnaMap = map[string]string{
	"u": "a",
	"a": "u",
	"c": "g",
	"g": "c",
}

var mRnaMin1SlipperySites = []string{
	"uuuuuua",
	"uuuaaac",
}

var min1SlipperySites = []string{
	"tttaaac",
	"tttaaag",
	"tttaaat",
}

var dnaStartCodon = "atg"
var rnaStartCodon = "uac"
var mRnaStartCodon = "aug"

var dnaStopCodons = map[string]bool{
	"tag": true,
	"taa": true,
	"tga": true,
}

var mRnaStopCodons = map[string]bool{
	"auc": true,
	"auu": true,
	"acu": true,
}

var rnaStopCodons = map[string]bool{
	"uaa": true,
	"uag": true,
	"uga": true,
}

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

type Match struct {
	sequence    string
	sourceIndex int
	targetIndex int
	length      int
	distance    int
}

type Sequence struct {
	file        string
	baseName    string
	name        string
	suffix      string
	description string
	chars       string
	mRna        string
	rna         string
	protein     string // string polypeptides ( = string peptides ( = string amino acids))
	matches     []Match
	matchMask   string
	segments    []SequenceSegment
	segmentMask string
	compare     *Sequence
	charSet     []string
	labels      []string
}

type SequenceSegment struct {
	key   string
	start int
	end   int
}

func NewSequence() *Sequence {
	return &Sequence{}
}

func NewSequenceFromFile(file string) *Sequence {
	sequence := NewSequence()
	sequence.file = file
	sequence.baseName = filepath.Base(strings.TrimSuffix(file, filepath.Ext(file)))
	sequence.suffix = "-RNA"
	sequence.name = strings.Split(sequence.baseName, ".")[0]
	sequence.description = "RNA"
	sequence.appendSequenceFromFile(file)
	sequence.segmentMask = strings.Repeat("0", len(sequence.chars))
	sequence.matchMask = strings.Repeat("0", len(sequence.chars))
	return sequence
}

func readFileContent(file string) string {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
}

func (sequence *Sequence) appendSequenceFromFile(file string) {
	sequence.charSet = getDnaCharSet()
	if sequence.segments == nil {
		sequence.segments = []SequenceSegment{}
	}

	input := readFileContent(file)
	scanner := bufio.NewScanner(strings.NewReader(input))
	r1, _ := regexp.Compile("^\\s*(CDS|5'UTR)\\s+(.*)$")
	r2, _ := regexp.Compile("([0-9]+)\\.\\.([0-9]+)")

	var chars string
	var length = int64(len(sequence.chars))

	for scanner.Scan() {
		text := strings.Trim(scanner.Text(), " ")
		if text == "ORIGIN" {
			break
		}
		if !r1.MatchString(text) {
			continue
		}
		matches := r2.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			start, _ := strconv.ParseInt(match[1], 10, 0)
			start = length + start - 1
			end, _ := strconv.ParseInt(match[2], 10, 0)
			end = length + end - 1
			sequence.segments = append(sequence.segments, SequenceSegment{"", int(start), int(end)})
		}
	}

	for scanner.Scan() {
		text := strings.Trim(scanner.Text(), " ")
		if text == "//" {
			break
		}
		pieces := strings.Split(text, " ")
		chars += strings.Join(pieces[1:len(pieces)], "")
	}

	sequence.chars += chars
}

func (sequence *Sequence) findOrfs() {
	for i := 0; i < len(sequence.chars); {
		remainder := sequence.chars[i:]
		start := strings.Index(remainder, dnaStartCodon)
		if start == -1 {
			break
		}
		newSegment := remainder[start:]
		j := 0
		var aminoAcid string
		for ; j < len(newSegment)-2; j += 3 {
			codon := newSegment[j : j+3]
			if codon == dnaStartCodon {
				fmt.Printf("Found start codon at %d\n", i+start+j)
			}
			char := codonTranslation[strings.ToUpper(codon)]
			aminoAcid += string(char)
			if j >= 6 {
				last7 := newSegment[j-4 : j+3]
				if last7 == "tttaaac" ||
					last7 == "tttaaag" ||
					last7 == "tttaaat" {
					fmt.Printf("Found slippery site at %d %s\n", i+start+j, last7)
					j--
					continue
				}
			}
			if codon == "tag" ||
				codon == "taa" ||
				codon == "tga" {
				fmt.Printf("ORF at %d..%d: %s\n", i+start, i+start+j, aminoAcid)
				break
			}
		}
		i = i + start + j + 3
	}
}

func findLongestMatch(source string, target string, min int) Match {
	var match Match
	length := int(math.Min(float64(min), float64(len(source))))
	for ; (length) <= len(source); length++ {
		needle := source[0:length]
		if targetIndex := strings.Index(target, needle); targetIndex >= 0 {
			match = Match{needle, 0, targetIndex, length, 0}
			continue
		}
		break
	}
	return match
}

func findMatches(source string, target string, min int) (matches []Match, mask string) {
	mask = ""
	matches = []Match{}
	for i := 0; i < len(source); {
		match := findLongestMatch(source[i:], target, min)
		match.sourceIndex = i
		if match.length >= min {
			matches = append(matches, match)
			mask += strings.Repeat("1", match.length)
			i += match.length
			continue
		}
		mask += "0"
		i++
	}
	return matches, mask
}

func (sequence *Sequence) findMatches(compareSequence *Sequence, min int) {
	matches, mask := findMatches(sequence.chars, compareSequence.chars, min)
	sequence.matches = matches
	sequence.matchMask = mask
}

func (sequence *Sequence) transcribe() *Sequence {
	protein := NewSequence()
	protein.baseName = sequence.baseName
	protein.suffix = "-AA"
	protein.name = sequence.name
	protein.description = "Amino Acid"
	protein.charSet = getAminoAcidCharSet()
	protein.segments = []SequenceSegment{}
	// proteinChars := ""
	mRna := ""
	rna := ""
	for _, char := range sequence.chars {
		mRnaChar := dnaMRnaMap[string(char)]
		mRna += mRnaChar
		rna += mRnaRnaMap[mRnaChar]
	}
	for _, segment := range sequence.segments {
		segmentChars := sequence.chars[segment.start:segment.end]
		start := len(protein.chars)
		// mRnaSegment := mRna[segment.start:segment.end]
		// rnaSegment := rna[segment.start:segment.end]
		// fmt.Printf("Got segment %d..%d \n%s\n%s\n%s\n", segment.start, segment.end, dnaSegment, mRnaSegment, rnaSegment)
		// protein += "\n\n"
		for i := 0; i < len(segmentChars)-2; i += 3 {
			codon := segmentChars[i : i+3]
			// if codon == dnaStartCodon {
			// 	fmt.Printf("Got start at pos %d\n", i)
			// }
			char := codonTranslation[strings.ToUpper(codon)]
			protein.chars += string(char)
			// if _, ok := dnaStopCodons[codon]; ok {
			// 	fmt.Printf("Got stop at pos %d.\n", i)
			// }
		}
		protein.segments = append(protein.segments, SequenceSegment{segment.key, start, len(protein.chars)})
	}
	sequence.mRna = mRna
	sequence.rna = rna
	// dnaSequence.protein = protein
	// dnaSequence.proteinSegments = proteinSegments
	// fmt.Printf("%s\n", protein.chars)
	return protein
}

func replaceChars(sequence string, replacement string, offset int) string {
	return sequence[:offset] + replacement + sequence[offset+len(replacement):]
}

func createSegmentMask(sequence string, segments []SequenceSegment) string {
	count := 1
	mask := strings.Repeat("0", len(sequence))
	for _, segment := range segments {
		char := strconv.FormatInt(int64(count%36), 36)
		segmentChars := strings.Repeat(char, segment.end-segment.start)
		mask = mask[:segment.start] + segmentChars + mask[segment.end:]
		count++
	}
	return mask
}

func (sequence *Sequence) createSegmentMask() {
	sequence.segmentMask = createSegmentMask(sequence.chars, sequence.segments)
}

func (sequence *Sequence) getTotalMatchSize() int {
	var size int
	for _, match := range sequence.matches {
		size += match.length
	}
	return size
}

func (sequence *Sequence) getLongestMatch() int {
	var size int
	for _, match := range sequence.matches {
		size = int(math.Max(float64(size), float64(match.length)))
	}
	return size
}

func addLabel(img *image.RGBA, text string, x float64, y float64) {
	dc := gg.NewContextForRGBA(img)
	font, _ := truetype.Parse(gobold.TTF)
	// font, _ := truetype.Parse(goregular.TTF)
	// font, _ := truetype.Parse(gomono.TTF)
	// font, _ := truetype.Parse(gomonobold.TTF)
	face := truetype.NewFace(font, &truetype.Options{
		Size: 14,
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

func getSequencePalette(colors []RGBA, charSet []string, low float64, high float64) map[int]map[string]*RGBA {
	palette := map[int]map[string]*RGBA{}
	for i, color := range colors {
		for j, char := range charSet {
			if _, ok := palette[i]; !ok {
				palette[i] = map[string]*RGBA{}
			}
			luminance := float64(j) / float64(len(charSet))
			luminance = low + (luminance * (1 - low))
			luminance = luminance * high
			palette[i][char] = color.luminance(luminance)
		}
	}
	return palette
}

func getSequenceImg(sequence string, matchMask string, segmentMask string, charSet []string) *image.RGBA {
	charSetString := strings.Join(charSet, "")
	palette := getSequencePalette(getColors(), charSet, 0, 0.6)
	matchPalette := getSequencePalette(getMatchColors(), charSet, 0.6, 1)

	total := float64(len(sequence))
	// width := int(math.Ceil(math.Sqrt(total)))
	// height := width
	// total := width * width
	height := int(math.Ceil(math.Ceil(math.Sqrt(total))/float64(90)) * 90) // 100
	width := int(math.Ceil(total / float64(height)))
	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})
	dc := gg.NewContextForRGBA(img)
	dc.Clear()
	dc.SetRGB(0, 0, 0)

	var x, y int
	for i, b := range sequence {
		char := string(b)
		segmentMaskIndex, _ := strconv.ParseInt(string(segmentMask[i]), 36, 0)
		segmentMaskIndex = segmentMaskIndex % int64(len(palette))

		if strings.Index(charSetString, char) >= 0 {
			if len(matchMask) > i && string(matchMask[i]) == "1" {
				img.Set(x, y, matchPalette[int(segmentMaskIndex)][char])
			} else {
				img.Set(x, y, palette[int(segmentMaskIndex)][char])
			}
		}
		x++
		if x >= width {
			y++
			x = 0
		}
	}

	return img
}

func (sequence *Sequence) writeImage() {
	sequence.createSegmentMask()
	img := getSequenceImg(sequence.chars, sequence.matchMask, sequence.segmentMask, sequence.charSet)

	sb := img.Bounds()

	scaleFactor := int(math.Ceil(720 / float64(sb.Dy()))) // 100
	// scaleFactor := 1

	resizedImg := image.NewRGBA(image.Rect(0, 0, sb.Dx()*scaleFactor, sb.Dy()*scaleFactor))
	draw.NearestNeighbor.Scale(resizedImg, resizedImg.Bounds(), img, sb, draw.Over, nil)

	baseName := sequence.baseName
	description := sequence.description + " Sequence"
	if sequence.compare != nil && sequence.compare.baseName != "" {
		baseName = sequence.baseName + "-" + sequence.compare.baseName
		description = sequence.description + " Comparison"
	}

	addLabel(resizedImg, fmt.Sprintf("%s", description), 10, 20)
	addLabel(resizedImg, fmt.Sprintf("%s (%d codes)", sequence.name, len(sequence.chars)), 10, 40)

	for i, label := range sequence.labels {
		addLabel(resizedImg, label, 10, (60 + (float64(i) * 20)))
	}

	filePath := "./images/" + baseName + sequence.suffix + ".png"
	fmt.Printf("Writing image: %s\n", filePath)
	f, _ := os.Create(filePath)
	png.Encode(f, resizedImg)
}

func (sequence *Sequence) compareTo(compareSequence *Sequence, min int) {
	random := NewSequence()
	random.chars = StringWithCharset(len(compareSequence.chars), strings.Join(compareSequence.charSet, ""))
	sequence.findMatches(random, min)
	randomMatchSize := sequence.getTotalMatchSize()

	sequence.compare = compareSequence
	sequence.findMatches(compareSequence, min)
	totalMatchSize := sequence.getTotalMatchSize()

	sequence.labels = []string{
		fmt.Sprintf("%s (%d codes)", sequence.compare.name, len(sequence.compare.chars)),
		fmt.Sprintf("Match: %.2f%% (%d codes)", (float64(totalMatchSize) / float64(len(sequence.chars)) * float64(100)), totalMatchSize),
		fmt.Sprintf("Longest: %d codes", sequence.getLongestMatch()),
		fmt.Sprintf("Control (Random data): %.2f%%", (float64(randomMatchSize) / float64(len(sequence.chars)) * float64(100))),
	}

	fmt.Printf("\n%s\n", sequence.description)
	fmt.Printf("%s (%d codes)\n", sequence.name, len(sequence.chars))
	for _, label := range sequence.labels {
		fmt.Printf("%s\n", label)
	}
	fmt.Printf("Check RNA size: %d\n", len(sequence.chars))
	fmt.Printf("Check match size: %d\n", len(sequence.matchMask))
	fmt.Printf("Check segment size: %d\n", len(sequence.segmentMask))
}

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func (sequence1 *Sequence) writeCompareImages(sequence2 *Sequence, min int) {
	sequence1.compareTo(sequence2, min)
	sequence1.writeImage()

	sequence2.compareTo(sequence1, min)
	sequence2.writeImage()
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

		genome1 := NewSequenceFromFile("./examples/COVID-19.NC_045512.txt")

		// genome1.findOrfs()
		// os.Exit(0)

		genome1.writeImage()
		protein1 := genome1.transcribe()
		protein1.writeImage()

		compareFiles := []string{
			// "./examples/COVID-19-Peru.MT263074.txt",
			"./examples/RaTG13.MN996532.txt",
			"./examples/Pangolin-CoV.MT072864.txt",
			"./examples/HIV-1.AF033819.txt",
			"./examples/HIV-2.KU179861.txt",
			"./examples/SARS-CoV1.NC_004718.txt",
			"./examples/SARS-CoV1.AY278741.txt",
			"./examples/MERS-CoV.KT029139.txt",
			"./examples/HCoV-OC43.AY391777.txt",
			"./examples/HCoV-229E.MF542265.txt",
			"./examples/HCoV-NL63.MG772808.txt",
			"./examples/HCoV-HKU1.AY597011.txt",
			"./examples/EBOLA.NC_002549.txt",
			"./examples/HEP-C.NC_004102.txt",
			"./examples/Maesles.NC_001498.txt",
			"./examples/Rabies.NC_001542.txt",
			"./examples/COVID-19.MN908947.txt",
		}

		for _, compareFile := range compareFiles {
			func() {
				genome2 := NewSequenceFromFile(compareFile)
				genome1.writeCompareImages(genome2, 12)              // 10
				protein1.writeCompareImages(genome2.transcribe(), 5) // 4
			}()
		}

		// func() {
		// 	genome2 := NewSequence()
		// 	genome2.description = "RNA Sequence"
		// 	genome2.charSet = getDnaCharSet()
		// 	genome2.chars = StringWithCharset(len(genome1.chars), "atgc")
		// 	genome2.baseName = fmt.Sprintf("Random-%d", len(genome2.chars))
		// 	genome2.name = "Random RNA"
		// 	genome2.suffix = "-R"
		// 	genome1.writeCompareImages(genome2, 10)
		// 	protein2 := NewSequence()
		// 	protein2.description = "Amino Acid Sequence"
		// 	protein2.charSet = getAminoAcidCharSet()
		// 	protein2.chars = StringWithCharset(len(protein1.chars), strings.Join(protein1.charSet, ""))
		// 	protein2.baseName = fmt.Sprintf("Random-%d", len(protein2.chars))
		// 	protein2.name = "Random Amino Acid"
		// 	protein2.suffix = "-P"
		// 	protein1.writeCompareImages(protein2, 4)
		// }()

		os.Exit(0)

		func() {
			genome2 := NewSequenceFromFile("./examples/H1N1/H1N1-seg1-NC_026438.txt")
			genome2.appendSequenceFromFile("./examples/H1N1/H1N1-seg2-NC_026435.txt")
			genome2.appendSequenceFromFile("./examples/H1N1/H1N1-seg3-NC_026437.txt")
			genome2.appendSequenceFromFile("./examples/H1N1/H1N1-seg4-NC_026433.txt")
			genome2.appendSequenceFromFile("./examples/H1N1/H1N1-seg5-NC_026436.txt")
			genome2.appendSequenceFromFile("./examples/H1N1/H1N1-seg6-NC_026434.txt")
			genome2.appendSequenceFromFile("./examples/H1N1/H1N1-seg7-NC_026431.txt")
			genome2.appendSequenceFromFile("./examples/H1N1/H1N1-seg8-NC_026432.txt")
			genome1.compareTo(genome2, 9)
			genome1.writeImage()
			genome2.createSegmentMask()
			genome2.compareTo(genome1, 9)
			genome2.writeImage()
		}()
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
