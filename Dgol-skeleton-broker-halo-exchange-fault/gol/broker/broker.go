package main

import (
	"errors"
	"flag"
	"net"
	"time"

	//"flag"
	"fmt"
	//"log"
	//"net"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/gol/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type ConnectionBroker struct {
	aliveCells  []util.Cell //slice of aliveCells that can be shared by GoLmethod and AliveCellsreport
	currentTurn int         //current turn of gol
	mu          sync.Mutex  //mutex lock to protect critical section (reading and writing to alivecells)
}

var (
	globalWorld   [][]byte
	terminate     = false
	paused        = false
	shutDown      = make(chan bool)
	golFinished   = make(chan bool)
	aliveWorkers  = make([]bool, 4)
	clientsGlobal = make([]*rpc.Client, 4)
	mu            sync.Mutex
	initialClient *rpc.Client
)

func (c *ConnectionBroker) AliveCellsMethod(req stubs.Request, res *stubs.Response) (err error) {
	c.mu.Lock()
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	return
}

func (c *ConnectionBroker) SaveWorld(request stubs.Request, res *stubs.Response) (err error) {
	c.mu.Lock()
	mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()
	return
}

func (c *ConnectionBroker) TerminateWorld(request stubs.Request, res *stubs.Response) (err error) {
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

func (c *ConnectionBroker) PauseWorld(request stubs.Request, res *stubs.Response) (err error) {
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
}

func (c *ConnectionBroker) ResetMethod(request stubs.Request, res *stubs.Response) (err error) {
	c.mu.Lock()
	c.currentTurn = 0
	c.aliveCells = nil
	c.mu.Unlock()

	mu.Lock()
	globalWorld = nil
	terminate = false
	paused = false
	mu.Unlock()
	return
}

func (c *ConnectionBroker) TerminateClient(request stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()
	terminate = true
	c.mu.Lock()
	res.FinalWorld = globalWorld
	res.AliveCells = c.aliveCells
	res.CurrentTurn = c.currentTurn
	c.mu.Unlock()
	mu.Unlock()

	var req stubs.Request
	var resTemp stubs.Response

	initialClient.Go(stubs.TerminateWorker, req, &resTemp, nil)
	clientsGlobal[0].Go(stubs.TerminateWorker, req, &resTemp, nil)
	clientsGlobal[1].Go(stubs.TerminateWorker, req, &resTemp, nil)
	clientsGlobal[2].Go(stubs.TerminateWorker, req, &resTemp, nil)
	clientsGlobal[3].Go(stubs.TerminateWorker, req, &resTemp, nil)

	select {
	case shutDown <- true:

	default:
	}

	return
}

func HeartbeatMonitor(workerIndex int, client *rpc.Client) {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			var resHeartbeat stubs.Heartbeat

			err := client.Call(stubs.HeartbeatWorker, stubs.Request{}, &resHeartbeat)
			if err != nil {
				fmt.Println(err)
				mu.Lock()
				aliveWorkers[workerIndex] = false
				mu.Unlock()
			} else {
				mu.Lock()
				aliveWorkers[workerIndex] = resHeartbeat.Alive
				mu.Unlock()
			}
			if !resHeartbeat.Alive {
				return
			}
		}
	}()
}

func replaceWorker(workerIndex int) *rpc.Client {
	ports := []string{"8050", "8060", "8070", "8080"}
	addr := "localhost:" + ports[workerIndex]

	newWorker, err := rpc.Dial("tcp", addr)
	if err != nil {
		fmt.Println(err, "error redialing")
	}
	mu.Lock()
	aliveWorkers[workerIndex] = true
	clientsGlobal[workerIndex] = newWorker
	mu.Unlock()
	HeartbeatMonitor(workerIndex, newWorker)
	fmt.Println("worker", workerIndex, "redialed")
	return newWorker
}

func (c *ConnectionBroker) ParallelGolMethod(req stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()
	initialClient, err = rpc.Dial("tcp", "localhost:8040")
	if err != nil {
		fmt.Println(err, "Failed to connect to initial server")
	}
	mu.Unlock()

	c.mu.Lock()
	c.currentTurn = 0
	c.aliveCells = make([]util.Cell, 0)
	c.mu.Unlock()

	height := req.Height
	width := req.Width
	threads := 4
	rows := height / 4

	responses := make([]*stubs.Response, 4)
	done := make(chan *rpc.Call, 4)

	currentWorld := req.StartWorld
	mu.Lock()
	globalWorld = currentWorld
	mu.Unlock()

	var aliveCells []util.Cell
	err = initialClient.Call(stubs.AliveCells, req, &res)
	if err != nil {
		return err
	}
	aliveCells = res.AliveCells

	c.mu.Lock()
	c.aliveCells = aliveCells
	c.mu.Unlock()

	clients := make([]*rpc.Client, 4)

	client0, err := rpc.Dial("tcp", "localhost:8050")
	if err != nil {
		fmt.Println(err)
	} else {
		clients[0] = client0
		mu.Lock()
		aliveWorkers[0] = true
		clientsGlobal[0] = client0
		mu.Unlock()
	}

	HeartbeatMonitor(0, client0)

	client1, err := rpc.Dial("tcp", "localhost:8060")
	if err != nil {
		fmt.Println(err)
	} else {
		clients[1] = client1
		mu.Lock()
		aliveWorkers[1] = true
		clientsGlobal[1] = client1
		mu.Unlock()
	}
	HeartbeatMonitor(1, client1)

	client2, err := rpc.Dial("tcp", "localhost:8070")
	if err != nil {
		fmt.Println(err)
	} else {
		clients[2] = client2
		mu.Lock()
		aliveWorkers[2] = true
		clientsGlobal[2] = client2
		mu.Unlock()
	}
	HeartbeatMonitor(2, client2)

	client3, err := rpc.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println(err)
	} else {
		clients[3] = client3
		mu.Lock()
		aliveWorkers[3] = true
		clientsGlobal[3] = client3
		mu.Unlock()
	}
	HeartbeatMonitor(3, client3)

loopTurns:
	for round := 0; round < req.Turns; round++ {

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

		calls := make([]*rpc.Call, threads)

		for i := 0; i < 4; i++ {

			startY := i * rows
			topHalo := startY
			if startY > 0 {
				topHalo--
			}
			endY := (i + 1) * rows

			if i == threads-1 {
				endY = height - 1
			}

			slice := make([][]byte, rows+2)
			if startY == 0 {
				slice[0] = currentWorld[height-1]
				copy(slice[1:], currentWorld[topHalo:endY+1])
			} else if endY == height-1 {
				slice[rows+1] = currentWorld[0]
				copy(slice[0:rows+1], currentWorld[topHalo:endY+1])
			} else {
				slice = currentWorld[topHalo : endY+1]
			}

			newReq := stubs.Request{
				StartWorld:  slice,
				StartY:      startY,
				EndY:        endY,
				Height:      len(slice),
				Width:       width,
				Turns:       req.Turns,
				CurrentTurn: round,
				Threads:     threads,
				WorldHeight: height,
				WorkerIndex: i,
			}

			mu.Lock()
			alive := aliveWorkers[i]
			mu.Unlock()

			if !alive {
				fmt.Println("worker", i, "is dead")
				time.Sleep(5 * time.Second)
				newClient := replaceWorker(i)
				clients[i] = newClient
			}

			var resTemp = new(stubs.Response)
			responses[i] = resTemp
			calls[i] = clients[i].Go(stubs.NextState, newReq, &resTemp, done)

		}

		errorFree := true
		for i := 0; i < 4; i++ {
			<-calls[i].Done

			if calls[i].Error != nil {
				errorFree = false
				fmt.Println("worker", i, "failed mid-turn")
				fmt.Println("failed at turn", round)
				mu.Lock()
				aliveWorkers[i] = false
				mu.Unlock()
			}
		}

		//fmt.Println("trying turn", round)
		if errorFree {
			//fmt.Println("executed", round)
			aliveCells = nil
			var nextWorld [][]byte
			for i := 0; i < 4; i++ {
				nextWorld = append(nextWorld, responses[i].FinalWorld...)
				aliveCells = append(aliveCells, responses[i].AliveCells...)
			}
			c.mu.Lock()
			c.aliveCells = aliveCells
			c.currentTurn++
			c.mu.Unlock()

			currentWorld = nextWorld
		} else {
			round--
		}

		mu.Lock()
		globalWorld = currentWorld
		mu.Unlock()

	}
	c.mu.Lock()
	res.FinalWorld = currentWorld
	res.CurrentTurn = c.currentTurn
	res.AliveCells = aliveCells
	c.mu.Unlock()

	mu.Lock()
	terminate = false
	paused = false
	mu.Unlock()

	return
}

func (c *ConnectionBroker) HaloGolMethod(req stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()
	initialClient, err = rpc.Dial("tcp", "localhost:8040")
	if err != nil {
		fmt.Println(err, "Failed to connect to initial server")
	}
	mu.Unlock()

	c.mu.Lock()
	c.currentTurn = 0
	c.aliveCells = make([]util.Cell, 0)
	c.mu.Unlock()

	height := req.Height
	width := req.Width
	threads := 4
	rows := height / 4

	responses := make([]*stubs.HaloRows, 4)
	done := make(chan *rpc.Call, 4)
	responsesWorld := make([]*stubs.Response, 4)

	currentWorld := req.StartWorld
	mu.Lock()
	globalWorld = currentWorld
	mu.Unlock()

	var aliveCells []util.Cell
	err = initialClient.Call(stubs.AliveCells, req, &res)
	if err != nil {
		return err
	}
	aliveCells = res.AliveCells

	c.mu.Lock()
	c.aliveCells = aliveCells
	c.mu.Unlock()

	if req.Turns == 0 {
		c.mu.Lock()
		res.CurrentTurn = 0
		res.FinalWorld = currentWorld
		res.AliveCells = aliveCells
		c.mu.Unlock()
		return
	}

	clients := make([]*rpc.Client, 4)

	client0, err := rpc.Dial("tcp", "localhost:8050")
	if err != nil {
		fmt.Println(err)
	} else {
		clients[0] = client0
		mu.Lock()
		aliveWorkers[0] = true
		clientsGlobal[0] = client0
		mu.Unlock()
	}

	client1, err := rpc.Dial("tcp", "localhost:8060")
	if err != nil {
		fmt.Println(err)
	} else {
		clients[1] = client1
		mu.Lock()
		aliveWorkers[1] = true
		clientsGlobal[1] = client1
		mu.Unlock()
	}

	client2, err := rpc.Dial("tcp", "localhost:8070")
	if err != nil {
		fmt.Println(err)
	} else {
		clients[2] = client2
		mu.Lock()
		aliveWorkers[2] = true
		clientsGlobal[2] = client2
		mu.Unlock()
	}

	client3, err := rpc.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println(err)
	} else {
		clients[3] = client3
		mu.Lock()
		aliveWorkers[3] = true
		clientsGlobal[3] = client3
		mu.Unlock()
	}

	calls := make([]*rpc.Call, threads)

	if req.Turns == 1 {
		for i := 0; i < threads; i++ {
			startY := i * rows
			topHalo := startY
			if startY > 0 {
				topHalo--
			}
			endY := (i + 1) * rows

			if i == threads-1 {
				endY = height - 1
			}

			slice := make([][]byte, rows+2)
			if startY == 0 {
				slice[0] = currentWorld[height-1]
				copy(slice[1:], currentWorld[topHalo:endY+1])
			} else if endY == height-1 {
				slice[rows+1] = currentWorld[0]
				copy(slice[0:rows+1], currentWorld[topHalo:endY+1])
			} else {
				slice = currentWorld[topHalo : endY+1]
			}

			newReq := stubs.Request{
				StartWorld:  slice,
				StartY:      startY,
				EndY:        endY,
				Height:      len(slice),
				Width:       width,
				Turns:       req.Turns,
				CurrentTurn: 1,
				Threads:     threads,
				WorldHeight: height,
				WorkerIndex: i,
			}
			var resTemp = new(stubs.Response)
			responsesWorld[i] = resTemp
			calls[i] = clients[i].Go(stubs.NextState, newReq, &resTemp, done)
		}

		for i := 0; i < threads; i++ {
			<-calls[i].Done
		}

		aliveCells = nil
		var nextWorld [][]byte
		for i := 0; i < 4; i++ {
			nextWorld = append(nextWorld, responsesWorld[i].FinalWorld...)
			aliveCells = append(aliveCells, responsesWorld[i].AliveCells...)
		}
		c.mu.Lock()
		c.aliveCells = aliveCells
		c.currentTurn++
		res.FinalWorld = nextWorld
		res.CurrentTurn = c.currentTurn
		res.AliveCells = aliveCells
		c.mu.Unlock()
		return
	}

	for i := 0; i < threads; i++ {
		startY := i * rows
		topHalo := startY
		if startY > 0 {
			topHalo--
		}
		endY := (i + 1) * rows

		if i == threads-1 {
			endY = height - 1
		}

		slice := make([][]byte, rows+2)
		if startY == 0 {
			slice[0] = currentWorld[height-1]
			copy(slice[1:], currentWorld[topHalo:endY+1])
		} else if endY == height-1 {
			slice[rows+1] = currentWorld[0]
			copy(slice[0:rows+1], currentWorld[topHalo:endY+1])
		} else {
			slice = currentWorld[topHalo : endY+1]
		}

		newReq := stubs.Request{
			StartWorld:  slice,
			StartY:      startY,
			EndY:        endY,
			Height:      len(slice),
			Width:       width,
			Turns:       req.Turns,
			CurrentTurn: 1,
			Threads:     threads,
			WorldHeight: height,
			WorkerIndex: i,
		}
		var haloRes = new(stubs.HaloRows)
		responses[i] = haloRes
		calls[i] = clients[i].Go(stubs.SendWorldGetHalos, newReq, &haloRes, done)
	}

	for i := 0; i < threads; i++ {
		<-calls[i].Done
	} // one turn executed

	responsesTemp := make([]*stubs.HaloRows, 4)

	for i := 0; i < 4; i++ {
		responsesTemp[i] = responses[i]
	}

	for round := 2; round < req.Turns; round++ {

		for i := 0; i < threads; i++ {

			startY := i * rows
			topHalo := startY
			if startY > 0 {
				topHalo--
			}
			endY := (i + 1) * rows

			if i == threads-1 {
				endY = height - 1
			}

			if i == 0 {
				haloReq := stubs.HaloReq{
					TopHalo:     responses[3].BottomHalo,
					BottomHalo:  responses[i+1].TopHalo,
					Height:      rows + 2,
					Width:       width,
					Threads:     threads,
					StartY:      startY,
					EndY:        endY,
					WorldHeight: height,
					CurrentTurn: round,
				}
				var haloRes = new(stubs.HaloRows)
				responsesTemp[i] = haloRes
				calls[i] = clients[i].Go(stubs.SendHalosGetHalos, haloReq, &haloRes, done)
			} else if i == 3 {
				haloReq := stubs.HaloReq{
					TopHalo:     responses[i-1].BottomHalo,
					BottomHalo:  responses[0].TopHalo,
					Height:      rows + 2,
					Width:       width,
					Threads:     threads,
					StartY:      startY,
					EndY:        endY,
					CurrentTurn: round,
					WorldHeight: height,
				}
				var haloRes = new(stubs.HaloRows)
				responsesTemp[i] = haloRes
				calls[i] = clients[i].Go(stubs.SendHalosGetHalos, haloReq, &haloRes, done)
			} else {
				haloReq := stubs.HaloReq{
					TopHalo:     responses[i-1].BottomHalo,
					BottomHalo:  responses[i+1].TopHalo,
					Height:      rows + 2,
					Width:       width,
					Threads:     threads,
					StartY:      startY,
					EndY:        endY,
					CurrentTurn: round,
					WorldHeight: height,
				}
				var haloRes = new(stubs.HaloRows)
				responsesTemp[i] = haloRes
				calls[i] = clients[i].Go(stubs.SendHalosGetHalos, haloReq, &haloRes, done)
			}
		}

		for i := 0; i < threads; i++ {
			<-calls[i].Done
		}

		for i := 0; i < threads; i++ {
			responses[i] = responsesTemp[i]
		}

	} // for N turns here we have executed all N-1 turns

	for i := 0; i < threads; i++ {

		startY := i * rows
		topHalo := startY
		if startY > 0 {
			topHalo--
		}
		endY := (i + 1) * rows

		if i == threads-1 {
			endY = height - 1
		}

		if i == 0 {
			haloReq := stubs.HaloReq{
				TopHalo:     responses[3].BottomHalo,
				BottomHalo:  responses[i+1].TopHalo,
				Height:      rows + 2,
				Width:       width,
				Threads:     threads,
				StartY:      startY,
				EndY:        endY,
				WorldHeight: height,
				CurrentTurn: req.Turns,
			}
			resTemp := new(stubs.Response)
			responsesWorld[i] = resTemp
			calls[i] = clients[i].Go(stubs.SendHalosGetWorld, haloReq, &resTemp, done)
		} else if i == 3 {
			haloReq := stubs.HaloReq{
				TopHalo:     responses[i-1].BottomHalo,
				BottomHalo:  responses[0].TopHalo,
				Height:      rows + 2,
				Width:       width,
				Threads:     threads,
				StartY:      startY,
				EndY:        endY,
				CurrentTurn: req.Turns,
				WorldHeight: height,
			}
			resTemp := new(stubs.Response)
			responsesWorld[i] = resTemp
			calls[i] = clients[i].Go(stubs.SendHalosGetWorld, haloReq, &resTemp, done)
		} else {
			haloReq := stubs.HaloReq{
				TopHalo:     responses[i-1].BottomHalo,
				BottomHalo:  responses[i+1].TopHalo,
				Height:      rows + 2,
				Width:       width,
				Threads:     threads,
				StartY:      startY,
				EndY:        endY,
				CurrentTurn: req.Turns,
				WorldHeight: height,
			}
			resTemp := new(stubs.Response)
			responsesWorld[i] = resTemp
			calls[i] = clients[i].Go(stubs.SendHalosGetWorld, haloReq, &resTemp, done)
		}
	}

	for i := 0; i < threads; i++ {
		<-calls[i].Done
	}

	aliveCells = nil
	var nextWorld [][]byte
	for i := 0; i < 4; i++ {
		nextWorld = append(nextWorld, responsesWorld[i].FinalWorld...)
		aliveCells = append(aliveCells, responsesWorld[i].AliveCells...)
	}
	c.mu.Lock()
	c.aliveCells = aliveCells
	c.currentTurn++
	res.FinalWorld = nextWorld
	res.CurrentTurn = c.currentTurn
	res.AliveCells = aliveCells
	c.mu.Unlock()
	return
}

func main() {
	err := rpc.Register(&ConnectionBroker{}) //registers Connection struct methods for RPC
	if err != nil {
		fmt.Println("rpc register error:", err)
	}
	pAddr := flag.String("port", "8030", "Port to listen on") //specify a port
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

	/*defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("listener close error:", err)
		}
	}(listener)
	fmt.Printf("Listening on port %s\n", *pAddr)
	rpc.Accept(listener) //blocking call constantly accepting incoming connections*/

	<-shutDown

	fmt.Println("Shutting down...")
	err1 := listener.Close()
	if err1 != nil {
		return
	}
	fmt.Println("Goodbye!")
}
