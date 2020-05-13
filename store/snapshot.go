package store

import "github.com/hashicorp/raft"

type snapshot struct {
	c Cacher
}

// Persist saves the FSM snapshot out to the given sink.
func (s *snapshot) Persist(sink raft.SnapshotSink) error {

	sinkWriteClose := func() error {
		snapshotBytes, err := s.c.Marshal()
		if err != nil {
			return err
		}

		if _, err := sink.Write(snapshotBytes); err != nil {
			return err
		}

		if err := sink.Close(); err != nil {
			return err
		}
		return nil
	}

	if err := sinkWriteClose(); err != nil {
		sink.Cancel()
		return err
	}

	return nil
}

func (f *snapshot) Release() {}
