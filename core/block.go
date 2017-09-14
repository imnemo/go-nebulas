// Copyright (C) 2017 go-nebulas authors
//
// This file is part of the go-nebulas library.
//
// the go-nebulas library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-nebulas library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-nebulas library.  If not, see <http://www.gnu.org/licenses/>.
//

package core

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/sha3"
)

/*
BlockHeader type.
*/
type BlockHeader struct {
	hash       string
	parentHash string
	nonce      uint64
	coinbase   *Address
	timestamp  time.Time
}

/*
Block type.
*/
type Block struct {
	header       *BlockHeader
	transactions Transactions

	previousBlock *Block
	nextBlock     *Block
}

func NewBlock(parentHash string, nonce uint64, coinbase *Address) *Block {
	block := &Block{
		header:       &BlockHeader{parentHash: parentHash, nonce: nonce, coinbase: coinbase},
		transactions: make(Transactions, 10, 20),
	}
	return block
}

func (block *Block) AddTransactions(txs ...*Transaction) *Block {
	// TODO: dedup the transaction from chain.
	block.transactions = append(block.transactions, txs...)
	return block
}

func (block *Block) Sign() *Block {
	block.header.timestamp = time.Now()
	block.header.hash = block.computeHash()
	return block
}

func (block *Block) Nonce() uint64 {
	return block.header.nonce
}

func (block *Block) SetNonce(nonce uint64) {
	block.header.nonce = nonce
}

func (block *Block) Hash() string {
	return block.header.hash
}

func (block *Block) ParentHash() string {
	return block.header.parentHash
}

func (block *Block) computeHash() string {
	h := sha3.New256()

	h.Write([]byte(block.header.parentHash))
	h.Write([]byte(block.header.coinbase.address))

	bytes := make([]byte, 256)
	binary.LittleEndian.PutUint64(bytes[0:], block.header.nonce)
	h.Write(bytes[:8])

	binary.LittleEndian.PutUint64(bytes[0:], uint64(block.header.timestamp.UnixNano()))
	h.Write(bytes[:8])

	result := h.Sum(nil)
	return hex.EncodeToString(result)
}

func (block *Block) String() string {
	return fmt.Sprintf("Block {hash:%s; parentHash:%s; nonce:%d}",
		block.header.hash,
		block.header.parentHash,
		block.header.nonce,
	)
}