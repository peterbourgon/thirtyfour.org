package main

import (
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

func main() {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	var (
		templateFile = fs.String("template-file", "index.template", "template file")
		imagePath    = fs.String("image-path", "public/img", "path to images")
		perPage      = fs.Int("per-page", 50, "images per page")
		outputPath   = fs.String("output-path", "public", "path to write output files")
		verbose      = fs.Bool("v", false, "verbose log output")
	)
	fs.Usage = usageFor(fs, "render [flags]")
	fs.Parse(os.Args[1:])
	log.SetFlags(0)

	debug := log.New(ioutil.Discard, "", 0)
	if *verbose {
		debug = log.New(os.Stderr, "", 0)
	}

	// Parse the template. If this fails, no point continuing.
	template, err := template.ParseFiles(*templateFile)
	if err != nil {
		log.Fatalf("error parsing template: %v", err)
	}

	// Discover all of the images.
	var originals []string
	filepath.Walk(*imagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.Contains(path, "thumbnail") {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".jpg", ".jpeg", ".png", ".gif":
			originals = append(originals, path)
		}
		return nil
	})
	if len(originals) <= 0 {
		log.Fatalf("no images found")
	}
	debug.Printf("original image count %d", len(originals))

	// Sort images by filename, biggest first.
	sort.Slice(originals, func(i, j int) bool { return originals[i] > originals[j] })
	debug.Printf("first image: %s", originals[0])

	// Create thumbnails.
	var (
		completed   []imageData
		resizeBegin = time.Now()
	)
	for _, original := range originals {
		input, err := os.Open(original)
		if err != nil {
			log.Fatalf("error opening original image %s: %v", original, err)
		}

		img, format, err := image.Decode(input)
		if err != nil {
			input.Close()
			log.Fatalf("error decoding original image %s: %v", original, err)
		}
		input.Close()

		resized := resize(img, 500, 0, lanczos)
		thumbnail := original[:len(original)-len(filepath.Ext(original))] + "_thumbnail" + filepath.Ext(original)
		if err := os.MkdirAll(filepath.Dir(thumbnail), 0777); err != nil {
			log.Fatalf("error creating directory for %s: %v", original, err)
		}

		output, err := os.Create(thumbnail)
		if err != nil {
			log.Fatalf("error creating thumbnail for %s: %v", original, err)
		}

		switch format {
		case "jpeg":
			err = jpeg.Encode(output, resized, nil)
		case "png":
			err = png.Encode(output, resized)
		default:
			err = fmt.Errorf("unsupported file type %q", format)
		}
		if err != nil {
			output.Close()
			log.Fatalf("error encoding thumbnail for %s: %v", original, err)
		}

		if err := output.Close(); err != nil {
			log.Fatalf("error closing thumbnail file for %s: %v", original, err)
		}

		completed = append(completed, imageData{
			Original:  filepath.Base(original),
			Thumbnail: filepath.Base(thumbnail),
			Width:     resized.Bounds().Dx(),
			Height:    resized.Bounds().Dy(),
		})
	}
	debug.Printf("resized %d in %s", len(completed), time.Since(resizeBegin))

	// Group image data into pages.
	var (
		pages   [][]imageData
		current []imageData
	)
	for _, d := range completed {
		current = append(current, d)
		if len(current) >= *perPage {
			pages = append(pages, current)
			current = []imageData{}
		}
	}
	if len(current) > 0 {
		pages = append(pages, current)
	}
	debug.Printf("%d per page, page count %d", *perPage, len(pages))

	// Render the pages.
	for i, images := range pages {
		data := map[string]interface{}{"Images": images}
		if (i + 1) < len(pages) {
			data["ShowNext"] = true
			data["Next"] = (i + 1)
		}
		if (i - 1) >= 0 {
			data["ShowPrev"] = true
			data["Prev"] = (i - 1)
		}

		outputFilename := filepath.Join(*outputPath, fmt.Sprint(i), "index.html")
		if err := os.MkdirAll(filepath.Dir(outputFilename), 0777); err != nil {
			log.Fatalf("error creating directory for page %d: %v", i, err)
		}

		outputFile, err := os.Create(outputFilename)
		if err != nil {
			log.Fatalf("error creating page %d: %v", i, err)
		}

		if err := template.Execute(outputFile, data); err != nil {
			outputFile.Close()
			log.Fatalf("error rendering page %d: %v", i, err)
		}

		if err := outputFile.Close(); err != nil {
			log.Fatalf("error closing page %d: %v", i, err)
		}

		debug.Printf("rendered %s", outputFilename)
	}

	// Special case.
	if _, err := exec.Command("cp",
		filepath.Join(*outputPath, "0", "index.html"),
		filepath.Join(*outputPath, "index.html"),
	).CombinedOutput(); err != nil {
		log.Fatalf("error creating main index file: %v", err)
	}
}

type imageData struct {
	Original  string
	Thumbnail string
	Width     int
	Height    int
}

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		fmt.Fprintf(os.Stdout, "USAGE\n")
		fmt.Fprintf(os.Stdout, "  %s\n", short)
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "FLAGS\n")
		tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			def := f.DefValue
			if def == "" {
				def = "..."
			}
			fmt.Fprintf(tw, "  -%s %s\t%s\n", f.Name, f.DefValue, f.Usage)
		})
		tw.Flush()
		fmt.Fprintf(os.Stderr, "\n")
	}
}
