package rubberhose

import (
	"errors"
	"io"
)

type Partition struct {
	blockSize int64
	blocks    []*Block
}

func NewPartition(blockSize int64, blocks []*Block) Partition {
	return Partition{blockSize: blockSize, blocks: blocks}
}

func (par Partition) ReadAt(p []byte, off int64) (int, error) {
	blockNum := off / par.blockSize
	blockOff := off % par.blockSize
	if blockNum > int64(len(par.blocks)) {
		return 0, io.EOF
	}
	originalLength := len(p)
	for len(p) > 0 {
		if int(blockNum) >= len(par.blocks) {
			return originalLength - len(p), io.EOF
		}
		read, err := par.blocks[blockNum].ReadAt(p, blockOff)
		if err != io.EOF {
			return originalLength - len(p), err
		}
		p = p[read:]
		blockOff = 0
		blockNum++
	}
	return originalLength, nil
}

func (par Partition) WriteAt(p []byte, off int64) (int, error) {
	blockNum := off / par.blockSize
	blockOff := off % par.blockSize
	if blockNum > int64(len(par.blocks)) {
		return 0, io.EOF
	}
	originalLength := len(p)
	for len(p) > 0 {
		if int(blockNum) >= len(par.blocks) {
			return originalLength - len(p), io.EOF
		}
		written, err := par.blocks[blockNum].WriteAt(p, blockOff)
		p = p[written:]
		if err != nil && err != io.ErrShortWrite {
			return originalLength - len(p), err
		}
		blockOff = 0
		blockNum++
	}
	return originalLength, nil
}

func (par Partition) GetDataSize() int64 {
	var size int64
	for _, b := range par.blocks {
		size += b.GetDataSize()
	}
	return size
}

func (par Partition) orderBlocks() error {
	blocks := par.blocks
	finished := make([]*Block, 0, len(par.blocks))
	for len(blocks) != 0 {
		blockToRemove := -1
		for i, b := range blocks {
			nextBlockNum, err := b.GetNextBlockID()
			if err != nil {
				return err
			}
			if len(finished) == 0 {
				if nextBlockNum != -1 {
					continue
				}
				finished = append(finished, b)
				blockToRemove = i
				break
			}
			lastBlockNum := finished[len(finished)-1].blockNum
			if nextBlockNum == lastBlockNum {
				finished = append(finished, b)
				blockToRemove = i
				break
			}
		}
		if blockToRemove == -1 {
			return errors.New("invalid block structure")
		}
		blocks[len(blocks)-1], blocks[blockToRemove] = blocks[blockToRemove], blocks[len(blocks)-1]
		blocks = blocks[:len(blocks)-1]
	}
	for i, j := 0, len(finished)-1; i < j; i, j = i+1, j-1 {
		finished[i], finished[j] = finished[j], finished[i]
	}
	copy(par.blocks, finished)
	return nil
}

func (par Partition) Close() error {
	return nil
}