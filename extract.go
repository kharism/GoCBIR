package main

import (
	"encoding/gob"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"
	"errors"
	//"flag"
	//"log"
	//"runtime/pprof"

	kdtree "github.com/kharism/go-kdtree"
	colorful "github.com/lucasb-eyer/go-colorful"
)

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
func (a *rgba) Normalize() {
	a.R = a.R * 255 / 65535
	a.G = a.G * 255 / 65535
	a.B = a.B * 255 / 65535
	//a.A = a.A * 255 / 65535
}

func (rgb *rgba) ToArray() []float64 {
	//c := colorful.Color{math.Sqrt(rgb.R), math.Sqrt(rgb.G), math.Sqrt(rgb.B)}
	//l, a, b := c.Lab()
	hasil := []float64{rgb.R, rgb.G, rgb.B}
	return hasil
}

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

type EuclideanPoint struct {
	kdtree.Point
	Filename string
	Vec      []float64
}

func (p EuclideanPoint) Dim() int {
	return len(p.Vec)
}

func (p EuclideanPoint) GetValue(dim int) float64 {
	return p.Vec[dim]
}

func (p EuclideanPoint) Distance(other kdtree.Point) float64 {
	var ret float64
	for i := 0; i < p.Dim(); i++ {
		tmp := p.GetValue(i) - other.GetValue(i)
		ret += tmp * tmp
	}
	return ret
}

func (p EuclideanPoint) PlaneDistance(val float64, dim int) float64 {
	tmp := p.GetValue(dim) - val
	return tmp * tmp
}
func NewEuclideanPoint(fileName string, vals ...float64) *EuclideanPoint {
	ret := &EuclideanPoint{}
	ret.Filename = fileName
	for _, val := range vals {
		ret.Vec = append(ret.Vec, val)
	}
	return ret
}

type DataPoint struct {
	Data     []colorful.Color
	Filename string
}

var WorkerNum = 4
//var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	/*flag.Parse()
	if *cpuprofile != "" {
        f, err := os.Create(*cpuprofile)
        if err != nil {
            log.Fatal(err)
        }
        pprof.StartCPUProfile(f)
        defer pprof.StopCPUProfile()
    }*/
	fi, _ := ioutil.ReadDir("data")
	var tree *kdtree.KDTree
	gob.Register(EuclideanPoint{})
	gob.Register(kdtree.KDTree{})
	gob.Register(kdtree.KdTreeNode{})
	points := []DataPoint{}
	if _, err := os.Stat("index.gob"); os.IsNotExist(err) {
		channelWorker := make(chan string, 400)
		channelJoiner := make(chan DataPoint, WorkerNum)
		wgExtractor := &sync.WaitGroup{}

		extractor := func() {
			for f := range channelWorker {
				data := GetFeature(f)
				newItem := DataPoint{data, f}
				channelJoiner <- newItem
			}
			wgExtractor.Done()
		}
		joiner := func() {
			for f := range channelJoiner {
				points = append(points, f)
			}
		}
		go joiner()
		for i := 0; i < WorkerNum; i++ {
			wgExtractor.Add(1)
			go extractor()
		}
		for _, f := range fi {
			//points = append(points)
			channelWorker <- "data/" + f.Name()
		}
		close(channelWorker)
		wgExtractor.Wait()
		close(channelJoiner)

		fmt.Println(len(points))
		buffer, err := os.Create("index.gob")
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		defer buffer.Close()
		enc := gob.NewEncoder(buffer)
		err = enc.Encode(&points)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

	} else {
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
	}
	//tree = kdtree.NewKDTree(points)

	fmt.Println(tree == nil)
	for i, targName := range os.Args {
		if i < 1 {
			continue
		}
		fmt.Println(targName)
		target := GetFeature(targName)
		now:=time.Now()
		neighbours := findKNN(target, 3, points)
		/*if reflect.TypeOf(neighbours[0]) == reflect.TypeOf(&EuclideanPoint{}) {
			for _, val := range neighbours {
				j := val.(*EuclideanPoint)
				fmt.Println(j.Filename)
				cmd := exec.Command("cp", j.Filename, ".")
				cmd.Run()
			}
		} else {
			for _, val := range neighbours {
				j := val.(EuclideanPoint)
				fmt.Println(j.Filename)
				cmd := exec.Command("cp", j.Filename, ".")
				cmd.Run()
			}
		}*/
		fmt.Println("NN Search",time.Since(now))
		for _, val := range neighbours {
			fmt.Println(val.filename, val.distance)
			cmd := exec.Command("cp", val.filename, ".")
			cmd.Run()
		}
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
	l1, a1, b1 := c1.R,c1.G,c1.B
	l2, a2, b2 := c2.R,c2.G,c2.B
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
type pair struct{
	X,Y 		[]colorful.Color
	Filename 	string
}
func findKNN(target []colorful.Color, numNeighbor int, db []DataPoint) []sortableStruct {
	best := bestResult{}
	best.best = []sortableStruct{}
	best.max = numNeighbor
	
	wg := sync.WaitGroup{}
	search := func(target []colorful.Color,db []DataPoint){
		subBest := bestResult{}
		subBest.best = []sortableStruct{}
		subBest.max = numNeighbor
		defer wg.Done()
		wg2 := sync.WaitGroup{}
		c := make(chan pair,400)
		for i:=0;i<Analyzer;i++{
			wg2.Add(1)
			go func(){
				for job:=range c{
					dist := HclDist(job.X, job.Y)//HclDist(db[i].Data, target)
					newItem := sortableStruct{}
					newItem.distance = dist
					newItem.data = job.X
					//fmt.Println(job.Filename)
					newItem.filename = job.Filename//db[i].Filename
					subBest.Lock()
					subBest.Add(newItem)
					subBest.Unlock()
				}
				wg2.Done()
			}()
		}
		for i := 0; i < len(db); i++ {
			c<-pair{db[i].Data,target,db[i].Filename}
		}
		close(c)
		wg2.Wait()
		best.Lock()
		for _,val := range subBest.best{
			best.Add(val)
		}
		best.Unlock()
	}
	for i:=0;i<Searcher;i++{
		wg.Add(1)
		if i<Searcher-1{
			fmt.Println(i*len(db)/Searcher,(i+1)*len(db)/Searcher)
			go search(target,db[i*len(db)/Searcher:(i+1)*len(db)/Searcher])
		}else{
			fmt.Println(i*len(db)/Searcher)
			go search(target,db[i*len(db)/Searcher:])
		}
		
	}
	wg.Wait()
	return best.best
}
