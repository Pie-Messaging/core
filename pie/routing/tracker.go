package routing

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/Pie-Messaging/core/pie"
	"math/big"
	"sync"
)

type Tracker struct {
	ID       *big.Int
	Addr     Addr
	session  *pie.Session
	LiveFlag int32
	mutex    sync.RWMutex
}

func (t *Tracker) Connect(ctx context.Context, protocol string, cert ...*tls.Certificate) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	tlsConfig := &tls.Config{
		NextProtos:         []string{protocol},
		InsecureSkipVerify: true,
	}
	session, err := pie.Connect(ctx, tlsConfig, t.Addr...)
	if err != nil {
		return err
	}
	t.session = session
	if protocol == pie.TrackerTLSProto {
		err := t.session.SendCert(cert[0])
		if err != nil {
			return err
		}
	}
	if t.ID.BitLen() == 0 {
		t.ID.SetBytes(t.session.GetPeerIDByCertHash())
	}
	return nil
}

func (t *Tracker) Session() *pie.Session {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.session
}

func (t *Tracker) SetAddrStr(addr string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	err := json.Unmarshal([]byte(addr), &t.Addr)
	if err != nil {
		pie.Logger.Println("Failed to unmarshal tracker address:", err)
		return err
	}
	return nil
}

func (t *Tracker) GetAddrStr() string {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	addr, err := json.Marshal(t.Addr)
	if err != nil {
		pie.Logger.Println("Failed to marshal tracker address:", err)
		return ""
	}
	return string(addr)
}
