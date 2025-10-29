package parser

import (
	"io"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/idursun/jjui/internal/screen"
)

type ControlMsg int

const (
	RequestMore ControlMsg = iota
	Close
)

type RowBatch struct {
	Rows    []Row
	HasMore bool
}

func ParseRowsStreaming(reader io.Reader, controlChannel <-chan ControlMsg, batchSize int) (<-chan RowBatch, error) {
	rowsChan := make(chan RowBatch, 1)
	go func() {
		defer close(rowsChan)
		var rows []Row
		var row Row
		rawSegments := screen.ParseFromReader(reader)
		for segmentedLine := range screen.BreakNewLinesIter(rawSegments) {
			rowLine := NewGraphRowLine(segmentedLine)
			if changeIdIdx := rowLine.FindPossibleChangeIdIdx(); changeIdIdx != -1 && changeIdIdx != len(rowLine.Segments)-1 {
				rowLine.Flags = Revision | Highlightable
				previousRow := row
				if len(rows) > batchSize {
					select {
					case msg := <-controlChannel:
						switch msg {
						case Close:
							return
						case RequestMore:
							rowsChan <- RowBatch{Rows: rows, HasMore: true}
							rows = nil
							break
						}
					}
				}
				row = NewGraphRow()
				if previousRow.Commit != nil {
					rows = append(rows, previousRow)
					row.Previous = &previousRow
				}
				for j := range changeIdIdx {
					row.Indent += utf8.RuneCountInString(rowLine.Segments[j].Text)
				}
				row.Commit.ChangeId = rowLine.Segments[changeIdIdx].Text
				fullChangeId := row.Commit.ChangeId
				for nextIdx := changeIdIdx + 1; nextIdx < len(rowLine.Segments); nextIdx++ {
					nextSegment := rowLine.Segments[nextIdx]
					if strings.TrimSpace(nextSegment.Text) == "" || strings.ContainsAny(nextSegment.Text, "\n\t\r ") {
						break
					}
					fullChangeId += nextSegment.Text
				}

				// get only if it contains conflicted "??" suffix
				if strings.HasSuffix(fullChangeId, "??") {
					row.Commit.ChangeId = fullChangeId
				}

				id, err := rowLine.GetCommitID()
				if err != nil {
					log.Printf("getting CommitID failed: %s", err)
				} else {
					row.Commit.CommitId = id
				}
			}
			row.AddLine(&rowLine)
		}
		if row.Commit != nil {
			rows = append(rows, row)
		}
		if len(rows) > 0 {
			select {
			case msg := <-controlChannel:
				switch msg {
				case Close:
					return
				case RequestMore:
					rowsChan <- RowBatch{Rows: rows, HasMore: false}
					rows = nil
					break
				}
			}
		}
		<-controlChannel
	}()
	return rowsChan, nil
}
