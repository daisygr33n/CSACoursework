package stubs

import "uk.ac.bris.cs/gameoflife/util"

var ExecGolMethod = "Connection.GolMethod"
var AliveCellsMethod = "Connection.AliveCellsMethod"

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
