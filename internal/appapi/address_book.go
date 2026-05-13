package appapi

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

type BindRequest struct {
	Address  string `json:"addr"`
	InviteID string `json:"invite_id"`
}

type BindResponse struct {
	UserID string `json:"user_id"`
}

type AddressBookCreateRequest struct {
	Address   string `json:"address"`
	Chain     string `json:"chain"`
	IsDefault bool   `json:"is_default"`
	Remark    string `json:"remark"`
}

type AddressBookUpdateRequest struct {
	ID        int64  `json:"id"`
	Address   string `json:"address"`
	Chain     string `json:"chain"`
	IsDefault bool   `json:"is_default"`
	Remark    string `json:"remark"`
}

type AddressBookDeleteRequest struct {
	ID int64 `json:"id"`
}

type AddressBookItem struct {
	ID        int64  `json:"id"`
	UserID    string `json:"user_id"`
	Address   string `json:"address"`
	Chain     string `json:"chain"`
	IsDefault bool   `json:"is_default"`
	Remark    string `json:"remark"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type AddressBookStore struct {
	mu     sync.Mutex
	nextID int64
	items  map[string][]AddressBookItem
}

func NewAddressBookStore() *AddressBookStore {
	return &AddressBookStore{
		nextID: 1,
		items:  make(map[string][]AddressBookItem),
	}
}

func UserIDForWallet(address string) string {
	normalized := strings.ToLower(strings.TrimSpace(address))
	hash := sha256.Sum256([]byte(normalized))
	return "local_" + hex.EncodeToString(hash[:8])
}

func (s *AddressBookStore) List(walletAddress string, chain string) []AddressBookItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	userID := UserIDForWallet(walletAddress)
	filterChain := normalizeChain(chain)
	list := make([]AddressBookItem, 0, len(s.items[userID]))
	for _, item := range s.items[userID] {
		if filterChain != "" && normalizeChain(item.Chain) != filterChain {
			continue
		}
		list = append(list, item)
	}

	return list
}

func (s *AddressBookStore) Create(walletAddress string, request AddressBookCreateRequest) (AddressBookItem, error) {
	if strings.TrimSpace(request.Address) == "" {
		return AddressBookItem{}, errors.New("address is required")
	}
	if strings.TrimSpace(request.Chain) == "" {
		return AddressBookItem{}, errors.New("chain is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	userID := UserIDForWallet(walletAddress)
	item := AddressBookItem{
		ID:        s.nextID,
		UserID:    userID,
		Address:   strings.TrimSpace(request.Address),
		Chain:     strings.TrimSpace(request.Chain),
		IsDefault: request.IsDefault,
		Remark:    strings.TrimSpace(request.Remark),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.nextID++

	if item.IsDefault {
		s.clearDefaultLocked(userID, item.Chain)
	}
	s.items[userID] = append(s.items[userID], item)

	return item, nil
}

func (s *AddressBookStore) Update(walletAddress string, request AddressBookUpdateRequest) (AddressBookItem, error) {
	if request.ID <= 0 {
		return AddressBookItem{}, errors.New("id is required")
	}
	if strings.TrimSpace(request.Address) == "" {
		return AddressBookItem{}, errors.New("address is required")
	}
	if strings.TrimSpace(request.Chain) == "" {
		return AddressBookItem{}, errors.New("chain is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	userID := UserIDForWallet(walletAddress)
	for index := range s.items[userID] {
		if s.items[userID][index].ID != request.ID {
			continue
		}

		if request.IsDefault {
			s.clearDefaultLocked(userID, request.Chain)
		}
		s.items[userID][index].Address = strings.TrimSpace(request.Address)
		s.items[userID][index].Chain = strings.TrimSpace(request.Chain)
		s.items[userID][index].IsDefault = request.IsDefault
		s.items[userID][index].Remark = strings.TrimSpace(request.Remark)
		s.items[userID][index].UpdatedAt = time.Now().Unix()

		return s.items[userID][index], nil
	}

	return AddressBookItem{}, errors.New("address book item not found")
}

func (s *AddressBookStore) Delete(walletAddress string, id int64) error {
	if id <= 0 {
		return errors.New("id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	userID := UserIDForWallet(walletAddress)
	for index := range s.items[userID] {
		if s.items[userID][index].ID != id {
			continue
		}
		s.items[userID] = append(s.items[userID][:index], s.items[userID][index+1:]...)
		return nil
	}

	return errors.New("address book item not found")
}

func (s *AddressBookStore) clearDefaultLocked(userID string, chain string) {
	normalized := normalizeChain(chain)
	for index := range s.items[userID] {
		if normalizeChain(s.items[userID][index].Chain) == normalized {
			s.items[userID][index].IsDefault = false
		}
	}
}
