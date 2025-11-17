package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/gol/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

// added variables to connection struct so both methods could access same variables
type Connection struct {
	aliveCells  []util.Cell //slice of aliveCells that can be shared by GoLmethod and AliveCellsreport
	currentTurn int         //current turn of gol
	mu          sync.Mutex  //mutex lock to protect critical section (reading and writing to alivecells)
}

var (
	globalWorld [][]byte
	terminate   = false
	paused      = false
	mu          sync.Mutex
)

var shutDown = make(chan bool)
var golFinished = make(chan bool)

// helper function to calculate alive cells surrounding the current cell
func calculateAliveNeighbours(world [][]byte, height, width int, x, y int) int {
	count := 0
	for dx := -1; dx <= 1; dx++ { //calculating neighbour coordinates
		for dy := -1; dy <= 1; dy++ {

			if dx == 0 && dy == 0 {
				continue // skip the cell itself
			}
			nx := (x + dx + width) % width
			//ny := (y + dy + height) % height
			ny := y + dy
			//if ny >= 0 && ny < height {
			// this is for wrap around at edges
			// all cells have 8 neighbours including corner cells
			if world[ny][nx] == 255 {
				count++
			}
			//}
		}
	}
	return count
}

// helper function to calculate slice of alive cells
func calculateAliveCells(world [][]byte, startY int, height int, width int) []util.Cell {
	var alive []util.Cell
	//fmt.Println(startY, height, "startY and height")
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			//fmt.Println("checking alive cells for", x, y)
			if world[y][x] == 255 {
				//fmt.Println("appending alive cells for", x, y+startY)
				alive = append(alive, util.Cell{X: x, Y: y + startY}) //slice containing all alive cells
			}
		}
	}
	return alive
}

func (c *Connection) TerminateWorker(request stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()
	terminate = true
	c.mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()

	select {
	case shutDown <- true:

	default:

	}

	return nil
}

func (c *Connection) HeartbeatWorker(request stubs.Request, res *stubs.Heartbeat) (err error) {
	res.Alive = true
	return
}

func workerNextState(currentWorld [][]byte, width, height, startY, endY int, out chan<- [][]byte) {
	chunk := NextState(currentWorld, startY, endY, height, width)
	out <- chunk
}

func (c *Connection) NextStateInit(req stubs.Request, res *stubs.Response) (err error) {
	//fmt.Println("received slice")

	currentWorld := req.StartWorld
	height := req.Height
	width := req.Width
	threads := req.Threads
	rows := (req.EndY - req.StartY) / threads

	outChannel := make([]chan [][]byte, threads)
	for i := range outChannel {
		outChannel[i] = make(chan [][]byte)
	}

	for i := 0; i < threads; i++ {
		startY := i*rows + 1   //+ req.StartY
		endY := (i+1)*rows + 1 //+ req.StartY

		if i == threads-1 {
			endY = height - 1
		}
		//fmt.Println("next state init startY", startY)
		//fmt.Println("next state init endY", endY)
		go workerNextState(currentWorld, width, height, startY, endY, outChannel[i])
	}

	nextWorld := make([][]byte, 0, req.EndY-req.StartY) // Initialises 2D slice with dimensions of the image
	for i := 0; i < threads; i++ {
		chunk := <-outChannel[i]
		nextWorld = append(nextWorld, chunk...)
	}

	// here we request for halo rows from other workers, join them together and then repeat the simulation
	res.FinalWorld = nextWorld
	if req.EndY == req.WorldHeight-1 {
		res.AliveCells = calculateAliveCells(res.FinalWorld, req.StartY, req.EndY-req.StartY+1, req.Width)
	} else {
		res.AliveCells = calculateAliveCells(res.FinalWorld, req.StartY, req.EndY-req.StartY, req.Width)
	}
	res.CurrentTurn = req.CurrentTurn
	return
}

func (c *Connection) AliveCells(req stubs.Request, res *stubs.Response) (err error) {
	res.AliveCells = calculateAliveCells(req.StartWorld, req.StartY, req.Height, req.Width)
	res.CurrentTurn = req.CurrentTurn
	res.FinalWorld = req.StartWorld
	return
}

func NextState(currentWorld [][]byte, startY, endY, height, width int) [][]byte {
	nextWorld := make([][]byte, endY-startY) // Initialises 2D slice with dimensions of the image
	for i := range nextWorld {
		nextWorld[i] = make([]byte, width)
	}
	indexNextWorld := 0
	for y := startY; y < endY; y++ {
		for x := 0; x < width; x++ {
			neighbours := calculateAliveNeighbours(currentWorld, height, width, x, y)
			if currentWorld[y][x] == 255 {
				if neighbours < 2 || neighbours > 3 {
					nextWorld[indexNextWorld][x] = 0
				} else {
					nextWorld[indexNextWorld][x] = 255
				}
			} else if neighbours == 3 {
				nextWorld[indexNextWorld][x] = 255
			} else {
				nextWorld[indexNextWorld][x] = 0
			}
		}
		indexNextWorld++
	}
	select {
	case <-golFinished:
	default:
		close(golFinished)
	}
	return nextWorld
}

// starts rpc server
func main() {
	err := rpc.Register(&Connection{}) //registers Connection struct mehtods for RPC
	if err != nil {
		fmt.Println("rpc register error:", err)
	}
	pAddr := flag.String("port", "8040", "Port to listen on") //specify a port
	flag.Parse()
	listener, err := net.Listen("tcp", ":"+*pAddr) //listen on tcp port
	if err != nil {
		fmt.Println(err)
	}

	defer listener.Close()
	fmt.Printf("Listening on port %s\n", *pAddr)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				fmt.Println("Error", err)
				continue
			}
			go rpc.ServeConn(conn)
		}
	}()

	<-shutDown

	fmt.Println("Shutting down...")
	err1 := listener.Close()
	if err1 != nil {
		return
	}
	fmt.Println("Goodbye!")

	/*defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("listener close error:", err)
		}
	}(listener)
	fmt.Printf("Listening on port %s\n", *pAddr)
	rpc.Accept(listener) //blocking call constantly accepting incoming connections*/
}
