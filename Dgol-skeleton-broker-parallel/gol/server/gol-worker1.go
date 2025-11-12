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
			ny := (y + dy + width) % width
			// this is for wrap around at edges
			// all cells have 8 neighbours including corner cells
			if world[ny][nx] == 255 {
				count++
			}
		}
	}
	return count
}

// helper function to calculate slice of alive cells
func calculateAliveCells(world [][]byte, startY int, height int, width int) []util.Cell {
	var alive []util.Cell
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if world[y][x] == 255 {
				alive = append(alive, util.Cell{X: x, Y: y + startY}) //slice containing all alive cells
			}
		}
	}
	return alive
}

/*func (c *Connection) SaveWorld(request stubs.Request, res *stubs.Response) (err error) {
	c.mu.Lock()
	mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()
	return
}

func (c *Connection) TerminateWorld(request stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()
	terminate = true
	c.mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()
	return
}

func (c *Connection) PauseWorld(request stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()

	if paused {
		paused = false

	} else {
		paused = true
	}

	c.mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()
	return
}*/

func (c *Connection) TerminateWorker(request stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()
	terminate = true
	c.mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()

	go func() {
		<-golFinished
		shutDown <- true
	}()

	return
}

// GolMethod executes gol for specified number of turns
/*func (c *Connection) GolMethod(req stubs.Request, res *stubs.Response) (err error) {

	currentWorld := req.StartWorld
	height := req.Height
	width := req.Width

	mu.Lock()
	globalWorld = currentWorld
	mu.Unlock()

	//computes alive cells for starting world
	alive := calculateAliveCells(currentWorld, 0, height, width)
	c.mu.Lock()                                                          //updating critical section so needs to be locked
	c.aliveCells = alive                                                 //update alivecells slice (Connection struct is shared)
	fmt.Println("Start of gol method, alive cells: ", len(c.aliveCells)) //debugging print statement
	c.currentTurn = 0                                                    //resets current turn to 0
	c.mu.Unlock()                                                        //unlock lock after finishing updating alive cells

	//main loop computing GOL
loopTurns:
	for rounds := 0; rounds < req.Turns; rounds++ { //loops for each turn in p.turns

		mu.Lock()
		if terminate {
			mu.Unlock()
			break loopTurns
		}
		mu.Unlock()

		mu.Lock()
		localPaused := paused
		mu.Unlock()

		if localPaused {
		loopPaused:
			for {
				mu.Lock()
				if !paused {
					mu.Unlock()
					break loopPaused
				}
				mu.Unlock()
			}
		}

		nextWorld := make([][]byte, height) // Initialises 2D slice with dimensions of the image
		for i := range nextWorld {
			nextWorld[i] = make([]byte, width)
		}

		for y := 0; y < height; y++ { //loops over each row and column checking if its neighbours are alive
			for x := 0; x < width; x++ {
				neighbours := calculateAliveNeighbours(currentWorld, height, width, x, y)
				if currentWorld[y][x] == 255 {
					if neighbours < 2 || neighbours > 3 {
						nextWorld[y][x] = 0
					} else {
						nextWorld[y][x] = 255 //gol logic
					}
				} else if neighbours == 3 {
					nextWorld[y][x] = 255
				} else {
					nextWorld[y][x] = 0
				}
			}
		}
		currentWorld = nextWorld //advance to next world
		mu.Lock()
		globalWorld = currentWorld
		mu.Unlock()

		//update alive cells and current turn
		alive := calculateAliveCells(currentWorld, 0, height, width)
		c.mu.Lock()          //lock during critical section
		c.aliveCells = alive //change shared section (Connection struct)
		c.currentTurn = rounds + 1
		fmt.Println("ALive cells in GOLMethod: ", len(alive)) //debugging statements
		fmt.Println("Current turn: ", c.currentTurn)
		c.mu.Unlock() //unlock after finishing editing critical sections
	}
	//send the response back to client
	c.mu.Lock()
	mu.Lock()
	res.FinalWorld = globalWorld
	res.CurrentTurn = c.currentTurn
	res.AliveCells = c.aliveCells
	terminate = false
	mu.Unlock()
	c.mu.Unlock()
	fmt.Println("returning from simulation")

	select {
	case <-golFinished:
	default:
		close(golFinished)
	}

	return
}*/

// returns current number of alive cells and turn for client (used for two second ticker)
/*func (c *Connection) AliveCellsMethod(req stubs.Request, res *stubs.Response) error {
	c.mu.Lock()                   //lock critical section
	res.AliveCells = c.aliveCells //accessing shared Connection struct
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock() //unlock
	return nil
}*/

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
		startY := i*rows + req.StartY
		endY := (i+1)*rows + req.StartY

		if i == threads-1 {
			endY = height
		}
		go workerNextState(currentWorld, width, height, startY, endY, outChannel[i])
	}

	nextWorld := make([][]byte, 0, req.EndY-req.StartY) // Initialises 2D slice with dimensions of the image
	for i := 0; i < threads; i++ {
		chunk := <-outChannel[i]
		nextWorld = append(nextWorld, chunk...)
	}

	//res.FinalWorld = NextState(req.StartWorld, req.StartY, req.EndY, req.Height, req.Width)
	res.FinalWorld = nextWorld
	//fmt.Println(len(calculateAliveCells(nextWorld, req.StartY, req.EndY-req.StartY, req.Width)))
	res.AliveCells = calculateAliveCells(res.FinalWorld, req.StartY, req.EndY-req.StartY, req.Width)
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
