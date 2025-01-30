package jobs

import (
	"log"
	"time"
)

type BatchProcessor struct {
	batchSize int
	processor func([]uint) error
}

func NewBatchProcessor(batchSize int, processor func([]uint) error) *BatchProcessor {
	return &BatchProcessor{
		batchSize: batchSize,
		processor: processor,
	}
}

func (bp *BatchProcessor) ProcessInBatches(userIDs []uint) error {
	for i := 0; i < len(userIDs); i += bp.batchSize {
		end := i + bp.batchSize
		if end > len(userIDs) {
			end = len(userIDs)
		}

		if err := bp.processor(userIDs[i:end]); err != nil {
			log.Printf("Error processing batch %d-%d: %v", i, end, err)
		}

		// Pequena pausa entre batches para não sobrecarregar o sistema
		time.Sleep(time.Second)
	}
	return nil
}
