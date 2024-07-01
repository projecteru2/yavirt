package local

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/store"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

// Snapshot .
// etcd keys:
//
//	/snapshots/<snap id>
type Snapshot struct {
	*meta.Generic

	BaseSnapshotID string `json:"base_snapshot"`
	Type           string `json:"type"`
	VolID          string `json:"vol"`
}

// LoadSnapshot .
func LoadSnapshot(id string) (*Snapshot, error) {
	s := NewSnapShot("")

	s.ID = id

	if err := meta.Load(s); err != nil {
		return nil, errors.Wrap(err, "")
	}

	return s, nil
}

// NewSnapShot .
func NewSnapShot(
	volID string,
) *Snapshot {
	var snap = newSnapshot()
	snap.VolID = volID

	return snap
}

func newSnapshot() *Snapshot {
	return &Snapshot{Generic: meta.NewGeneric()}
}

func (s *Snapshot) SetVolID(volID string) {
	s.VolID = volID
}

// Filepath .
func (s *Snapshot) Filepath() string {
	return s.JoinVirtPath(filepath.Join("snaps", s.Name()))
}

// Name .
func (s *Snapshot) Name() string {
	return fmt.Sprintf("%s-%s.snap", s.VolID, s.ID)
}

// Create .
func (s *Snapshot) Create() error {
	res := meta.Resources{s}

	return meta.Create(res)
}

// Save updates metadata to persistence store.
func (s *Snapshot) Save() error {
	return meta.Save(meta.Resources{s})
}

// Delete .
func (s *Snapshot) Delete() error {
	keys := []string{s.MetaKey()}
	vers := map[string]int64{s.MetaKey(): s.GetVer()}

	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	return store.Delete(ctx, keys, vers)
}

// MetaKey .
func (s *Snapshot) MetaKey() string {
	return meta.SnapshotKey(s.ID)
}

// GenerateID .
func (s *Snapshot) GenerateID() {
	s.genID()
}

func (s *Snapshot) genID() {
	s.ID = idgen.Next()
}

func (s *Snapshot) String() string {
	return fmt.Sprintf("%s, created at: %s", s.ID, time.Unix(s.CreatedTime, 0))
}

// Snapshots .
type Snapshots []*Snapshot

func (snaps *Snapshots) Append(snap ...*Snapshot) {
	*snaps = append(*snaps, snap...)
}

func (snaps Snapshots) IDs() []string {
	v := make([]string, len(snaps))
	for i, snap := range snaps {
		v[i] = snap.ID
	}
	return v
}

// LoadSnapshots .
func LoadSnapshots(ids []string) (snaps Snapshots, err error) {
	snaps = make(Snapshots, len(ids))

	for i, id := range ids {
		if snaps[i], err = LoadSnapshot(id); err != nil {
			return nil, errors.Wrap(err, "")
		}
	}

	return snaps, nil
}

// Len .
func (snaps Snapshots) Len() int {
	return len(snaps)
}

// Find .
func (snaps Snapshots) Find(snapID string) (*Snapshot, error) {
	for _, s := range snaps {
		if s.ID == snapID {
			return s, nil
		}
	}

	return nil, errors.Wrapf(terrors.ErrInvalidValue, "snapID %s not exists", snapID)
}
