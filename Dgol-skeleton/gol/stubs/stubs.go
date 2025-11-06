package stubs

import "uk.ac.bris.cs/gameoflife/util"

var ExecGolMethod = "Connection.GolMethod"
var AliveCellsMethod = "Connection.AliveCellsMethod"
var SaveWorld = "Connection.SaveWorld"
var TerminateWorld = "Connection.TerminateWorld"
var PauseWorld = "Connection.PauseWorld"

type Response struct {
	FinalWorld  [][]byte
	AliveCells  []util.Cell
	CurrentTurn int
}

type Request struct {
	StartWorld [][]byte
	Turns      int
	Height     int
	Width      int
}
