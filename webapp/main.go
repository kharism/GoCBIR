package main

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/draw"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	colorful "github.com/lucasb-eyer/go-colorful"
)

/*func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}*/

//var templates = template.Must(template.ParseFiles("default.html"))

type colorMap [][]colorful.Color

func (a colorMap) ToArray() []colorful.Color {
	p := []colorful.Color{}
	for _, v := range a {
		for _, c := range v {
			p = append(p, c)
		}
	}
	return p
}

type rgba struct {
	R, G, B float64
}

func (a *rgba) Add(R, G, B float64) {
	a.R += R //uint64(R)
	a.G += G //uint64(G)
	a.B += B //uint64(B)
	//a.A += uint64(A * A)
}
func (a *rgba) Divide(A float64) {
	a.R /= A //uint64()
	a.G /= A //uint64(A)
	a.B /= A //uint64(A)
	//a.A /= uint64(A)
}

const BLOCK_SIZE = 40

func GetFeature(filename string) []colorful.Color {
	r, _ := os.Open(filename)
	img, _, _ := image.Decode(r)

	size := img.Bounds()
	imgRGBA := image.NewRGBA(size)
	draw.Draw(imgRGBA, img.Bounds(), img, image.ZP, draw.Src)
	LayoutColors := make([][]colorful.Color, BLOCK_SIZE)
	for i := 0; i < BLOCK_SIZE; i++ {
		LayoutColors[i] = make([]colorful.Color, BLOCK_SIZE)
	}
	for i := 0; i < BLOCK_SIZE; i++ {
		for j := 0; j < BLOCK_SIZE; j++ {
			bounds := image.Rect(i*int(math.Ceil(float64(size.Dx())/BLOCK_SIZE)), j*int(math.Ceil(float64(size.Dy())/BLOCK_SIZE)), (i+1)*int(math.Ceil(float64(size.Dx())/BLOCK_SIZE)), (j+1)*int(math.Ceil(float64(size.Dy())/BLOCK_SIZE)))

			subImage := imgRGBA.SubImage(bounds)
			subBounds := subImage.Bounds()
			newRGBA := rgba{}
			count := 0
			for i1 := subBounds.Min.X; i1 < subBounds.Max.X; i1++ {
				for j1 := subBounds.Min.Y; j1 < subBounds.Max.Y; j1++ {
					count += 1
					c := colorful.MakeColor(subImage.At(i1, j1))

					newRGBA.Add(c.R*c.R, c.G*c.G, c.B*c.B)
				}
			}
			newRGBA.Divide(float64(count))
			//newRGBA.Normalize()
			if count > 0 {
				LayoutColors[i][j] = colorful.Color{newRGBA.R, newRGBA.G, newRGBA.B}
			}

		}
	}
	l := colorMap(LayoutColors)
	return l.ToArray()
}

func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	/*b, err := ioutil.ReadFile("default.html")
	if err!=nil{
        fmt.Println(err.Error())
    }
    w.Write(b)*/
    templates, err := template.New("default.html").Funcs(template.FuncMap{"loop":func()[]int{
        tt:=len(points)/10
        hasil := []int{}
        for i:=0;i<tt;i++{
            hasil = append(hasil,i)
        }
        return hasil
    }}).ParseFiles("default.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	err = templates.Execute(w,nil)
	
}
func AssetHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println(r.URL.Path)
	//f := strings.Split(r.URL.Path, "/")
	filename := "./" + r.URL.Path
	b, _ := ioutil.ReadFile(filename)
	w.Write(b)
}
func ImageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Path)
	//f := strings.Split(r.URL.Path, "/")
	filename := "." + r.URL.Path
	filename = strings.Replace(filename, "image", "data", -1)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err.Error())
	}
	w.Write(b)
}
func DataHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println(r.URL.Path)
	//f := strings.Split(r.URL.Path, "/")
	filename := "./" + r.URL.Path
	b, _ := ioutil.ReadFile(filename)
	w.Write(b)
}
func SimilarityHandler(w http.ResponseWriter, r *http.Request) {
	f := strings.Split(r.URL.Path, "/")
	imageId := f[len(f)-2]
	nearestNeighbor := f[len(f)-1]
	fmt.Println(imageId, nearestNeighbor)
	id, _ := strconv.Atoi(imageId)
	nn, _ := strconv.Atoi(nearestNeighbor)
	target := points[id]
	NN := findKNN(target.Data, nn, points)
	//fmt.Println(NN)
	fileName := []string{}
	for _, val := range NN {
		fileName = append(fileName, val.filename)
	}
	encoder := json.NewEncoder(w)
	encoder.Encode(fileName)
}
func PagingHandler(w http.ResponseWriter, r *http.Request){
    f := strings.Split(r.URL.Path, "/")
    page := f[len(f)-1]
    pageInt,_ := strconv.Atoi(page)
    pageInt *= 10
    pageMax := pageInt+10
    if pageMax > len(points){
        pageMax = len(points)
    }
    templates, err := template.ParseFiles("main.tplt")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
    for i:=pageInt;i<=pageMax;i++{
        points[i].Index = i
    }
	err = templates.ExecuteTemplate(w, "main.tplt", points[pageInt:pageMax])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
type DataPoint struct {
	Data     []colorful.Color
	Filename string
    Index    int
}

var points []DataPoint

func main() {
	http.HandleFunc("/", DefaultHandler)
	http.HandleFunc("/assets/", AssetHandler)
	http.HandleFunc("/image/", ImageHandler)
	http.HandleFunc("/data/", DataHandler)
	http.HandleFunc("/similar/", SimilarityHandler)
    http.HandleFunc("/paging/", PagingHandler)
	//http.HandleFunc("/save/", makeHandler(saveHandler))
	if _, err := os.Stat("index.gob"); os.IsNotExist(err) {
		fmt.Println("No Index file")
	} else {
        fmt.Println("Loading index")
		buffer, err := os.Open("index.gob")
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		defer buffer.Close()
		dec := gob.NewDecoder(buffer)
		err = dec.Decode(&points)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
        fmt.Println("Start Serving")
		http.ListenAndServe(":8080", nil)
	}

}

type sortableStruct struct {
	data     []colorful.Color
	filename string
	distance float64
}
type bestResult struct {
	sync.RWMutex
	best []sortableStruct
	max  int
}

func (b *bestResult) Add(s sortableStruct) {
	if len(b.best) < b.max {
		b.best = append(b.best, s)
	} else {
		if b.best[b.max-1].distance > s.distance {
			b.best = b.best[:b.max-1]
			b.best = append(b.best, s)
		}
	}
	sort.Slice(b.best, func(i, j int) bool {
		return b.best[i].distance < b.best[j].distance
	})
}
func sq(v float64) float64 {
	return v * v
}
func DistanceLab(c1, c2 colorful.Color) (float64, error) {
	l1, a1, b1 := c1.R, c1.G, c1.B
	l2, a2, b2 := c2.R, c2.G, c2.B
	hasil := sq(l1-l2) + sq(a1-a2) + sq(b1-b2)
	if math.IsNaN(hasil) {
		fmt.Println(l1, a1, b1, l2, a2, b2)
		return 0, errors.New("Nan Error")
	}
	return hasil, nil
}
func HclDist(a, b []colorful.Color) float64 {
	dist := 0.0

	for i := 0; i < len(a); i++ {
		//fmt.Println(a[i].R, a[i].G, a[i].B)

		l, e := DistanceLab(a[i], b[i])
		if e != nil {
			fmt.Println(a[i])
			os.Exit(-1)
		}
		dist += l
	}
	//fmt.Println(dist)
	return dist
}

var Searcher = 6
var Analyzer = 16

type pair struct {
	X, Y     []colorful.Color
	Filename string
}

func findKNN(target []colorful.Color, numNeighbor int, db []DataPoint) []sortableStruct {
	best := bestResult{}
	best.best = []sortableStruct{}
	best.max = numNeighbor

	wg := sync.WaitGroup{}
	search := func(target []colorful.Color, db []DataPoint) {
		subBest := bestResult{}
		subBest.best = []sortableStruct{}
		subBest.max = numNeighbor
		defer wg.Done()
		wg2 := sync.WaitGroup{}
		c := make(chan pair, 400)
		for i := 0; i < Analyzer; i++ {
			wg2.Add(1)
			go func() {
				for job := range c {
					dist := HclDist(job.X, job.Y) //HclDist(db[i].Data, target)
					newItem := sortableStruct{}
					newItem.distance = dist
					newItem.data = job.X
					//fmt.Println(job.Filename)
					newItem.filename = job.Filename //db[i].Filename
					subBest.Lock()
					subBest.Add(newItem)
					subBest.Unlock()
				}
				wg2.Done()
			}()
		}
		for i := 0; i < len(db); i++ {
			c <- pair{db[i].Data, target, db[i].Filename}
		}
		close(c)
		wg2.Wait()
		best.Lock()
		for _, val := range subBest.best {
			best.Add(val)
		}
		best.Unlock()
	}
	for i := 0; i < Searcher; i++ {
		wg.Add(1)
		if i < Searcher-1 {
			fmt.Println(i*len(db)/Searcher, (i+1)*len(db)/Searcher)
			go search(target, db[i*len(db)/Searcher:(i+1)*len(db)/Searcher])
		} else {
			fmt.Println(i * len(db) / Searcher)
			go search(target, db[i*len(db)/Searcher:])
		}

	}
	wg.Wait()
	return best.best
}
