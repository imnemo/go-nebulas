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
	"io/ioutil"

	"github.com/gogo/protobuf/proto"
	"github.com/nebulasio/go-nebulas/common/trie"
	"github.com/nebulasio/go-nebulas/core/pb"
	"github.com/nebulasio/go-nebulas/core/state"
	"github.com/nebulasio/go-nebulas/storage"
	"github.com/nebulasio/go-nebulas/util"
	log "github.com/sirupsen/logrus"
)

// Genesis Block Hash
var (
	GenesisHash      = make([]byte, BlockHashLength)
	GenesisTimestamp = int64(0)
)

// LoadGenesisConf load genesis conf for file
func LoadGenesisConf(filePath string) (*corepb.Genesis, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	content := string(b)

	genesis := new(corepb.Genesis)
	if err := proto.UnmarshalText(content, genesis); err != nil {
		return nil, err
	}
	return genesis, nil
}

// NewGenesisBlock create genesis @Block from file.
func NewGenesisBlock(conf *corepb.Genesis, chain *BlockChain) (*Block, error) {
	accState, err := state.NewAccountState(nil, chain.storage)
	if err != nil {
		return nil, err
	}
	txsTrie, err := trie.NewBatchTrie(nil, chain.storage)
	if err != nil {
		return nil, err
	}
	eventsTrie, err := trie.NewBatchTrie(nil, chain.storage)
	if err != nil {
		return nil, err
	}
	dposContext, err := NewDposContext(chain.storage)
	if err != nil {
		return nil, err
	}
	coinbase := &Address{make([]byte, AddressLength)}
	genesisBlock := &Block{
		header: &BlockHeader{
			chainID:     conf.Meta.ChainId,
			parentHash:  GenesisHash,
			dposContext: &corepb.DposContext{},
			coinbase:    coinbase,
			timestamp:   GenesisTimestamp,
			nonce:       0,
		},
		accState:    accState,
		txsTrie:     txsTrie,
		eventsTrie:  eventsTrie,
		dposContext: dposContext,
		txPool:      chain.txPool,
		storage:     chain.storage,
		height:      1,
		sealed:      false,
	}

	context, err := GenesisDynastyContext(chain.storage, conf)
	if err != nil {
		return nil, err
	}
	genesisBlock.LoadDynastyContext(context)
	genesisBlock.SetMiner(coinbase)

	genesisBlock.begin()
	// add token distribution for genesis
	for _, v := range conf.TokenDistribution {
		addr, err := AddressParse(v.Address)
		if err != nil {
			log.WithFields(log.Fields{
				"func":  "GenerateGenesisBlock",
				"block": v.Address,
				"err":   err,
			}).Error("wrong address in initial distribution.")
			return nil, err
		}
		acc := genesisBlock.accState.GetOrCreateUserAccount(addr.address)
		acc.AddBalance(util.NewUint128FromString(v.Value))
		log.Info(acc.Balance().Int64())
	}
	genesisBlock.commit()

	genesisBlock.Seal()
	genesisBlock.header.hash = GenesisHash
	return genesisBlock, nil
}

// CheckGenesisBlock if a block is a genesis block
func CheckGenesisBlock(block *Block) bool {
	if block == nil {
		return false
	}
	if block.Hash().Equals(GenesisHash) {
		return true
	}
	return false
}

// DumpGenesis return the configuration of the genesis block in the storage
func DumpGenesis(stor storage.Storage) (*corepb.Genesis, error) {
	genesis, err := LoadBlockFromStorage(GenesisHash, stor, nil, nil)
	if err != nil {
		return nil, err
	}
	dynasty, err := TraverseDynasty(genesis.dposContext.candidateTrie)
	if err != nil {
		return nil, err
	}
	bootstrap := []string{}
	for _, v := range dynasty {
		bootstrap = append(bootstrap, v.String())
	}
	distribution := []*corepb.GenesisTokenDistribution{}
	accounts, err := genesis.accState.Accounts()
	for _, v := range accounts {
		balance := v.Balance()
		if v.Address().Equals(genesis.Coinbase().Bytes()) {
			if v.Balance().Cmp(BlockReward.Int) == 0 {
				continue
			}
			balance = util.NewUint128FromBigInt(v.Balance().Sub(v.Balance().Int, BlockReward.Int))
		}
		distribution = append(distribution, &corepb.GenesisTokenDistribution{
			Address: string(v.Address().Hex()),
			Value:   balance.String(),
		})
	}
	return &corepb.Genesis{
		Meta: &corepb.GenesisMeta{ChainId: genesis.ChainID()},
		Consensus: &corepb.GenesisConsensus{
			Dpos: &corepb.GenesisConsensusDpos{Dynasty: bootstrap},
		},
		TokenDistribution: distribution,
	}, nil
}
