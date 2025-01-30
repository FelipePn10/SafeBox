package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type P2PStorage struct {
	host       host.Host
	peers      map[peer.ID]time.Time
	peersMutex sync.RWMutex
	baseDir    string
}

func NewP2PStorage(baseDir string) (*P2PStorage, error) {
	// Configurar host P2P
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.EnableRelay(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p2p host: %w", err)
	}

	// Configurar multiaddr para descoberta
	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 0))
	h.Network().Listen(addr)

	return &P2PStorage{
		host:    h,
		peers:   make(map[peer.ID]time.Time),
		baseDir: baseDir,
	}, nil
}

func (ps *P2PStorage) Save(ctx context.Context, file io.Reader, userID uint, fileName string) error {
	// Calcular hash do arquivo para deduplicação
	hash := sha256.New()
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	hash.Write(fileContent)
	fileHash := hex.EncodeToString(hash.Sum(nil))

	// Verificar se o arquivo já existe na rede P2P
	exists, err := ps.checkFileExists(ctx, fileHash)
	if err != nil {
		return err
	}

	if exists {
		// Se existe, apenas registrar o vínculo com o usuário
		return ps.linkFileToUser(ctx, userID, fileName, fileHash)
	}

	// Se não existe, distribuir para os peers
	return ps.distributeFile(ctx, fileContent, fileHash, userID, fileName)
}

func (ps *P2PStorage) GetTotalUsage(ctx context.Context, userID uint) (int64, error) {
	// Implementar cálculo de uso considerando deduplicação
	files, err := ps.getUserFiles(ctx, userID)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, file := range files {
		// Se o arquivo é compartilhado, conta apenas uma fração do tamanho
		if ps.isFileShared(file.Hash) {
			total += file.Size / int64(ps.getFileUsersCount(file.Hash))
		} else {
			total += file.Size
		}
	}

	return total, nil
}
