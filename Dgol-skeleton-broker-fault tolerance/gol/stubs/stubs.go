package stubs

import "uk.ac.bris.cs/gameoflife/util"

/*
var ExecGolMethod = "Connection.GolMethod"
var AliveCellsMethod = "Connection.AliveCellsMethod"
*/
var SaveWorld = "ConnectionBroker.SaveWorld"
var TerminateWorld = "ConnectionBroker.TerminateWorld"
var PauseWorld = "ConnectionBroker.PauseWorld"
var TerminateClient = "ConnectionBroker.TerminateClient"
var ResetMethod = "ConnectionBroker.ResetMethod"

// var NewMethod = "ConnectionBroker.ExecMethod"
var AliveCellsBrokers = "ConnectionBroker.AliveCellsMethod"
var NextState = "Connection.NextStateInit"
var ParallelGolMethod = "ConnectionBroker.ParallelGolMethod"
var AliveCells = "Connection.AliveCells"
var TerminateWorker = "Connection.TerminateWorker"
var HeartbeatWorker = "Connection.HeartbeatWorker"

type Response struct {
	FinalWorld  [][]byte
	AliveCells  []util.Cell
	CurrentTurn int
}

type Request struct {
	StartWorld  [][]byte
	Turns       int
	Height      int
	Width       int
	StartY      int
	EndY        int
	CurrentTurn int
	Threads     int
	WorldHeight int
	WorkerIndex int
}

type Heartbeat struct {
	Alive bool
}
