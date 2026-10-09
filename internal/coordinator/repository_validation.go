package coordinator

import (
	"encoding/json"
	"fmt"
	"github.com/regb/workitem/internal/model"
	bolt "go.etcd.io/bbolt"
)

func validateRepositoryReference(tx *bolt.Tx, m model.Manifest) error {
	repositories := tx.Bucket(bucketProjections).Bucket([]byte(RepositoryProjection))
	if m.Repository.ID == "" {
		if repositories != nil && repositories.Stats().KeyN > 0 {
			return fmt.Errorf("work items require a registered repository identity")
		}
		return nil
	}
	if repositories == nil {
		return fmt.Errorf("repository %s is not registered", m.Repository.ID)
	}
	raw := repositories.Get([]byte(m.Repository.ID))
	if raw == nil {
		return fmt.Errorf("repository %s is not registered", m.Repository.ID)
	}
	var r model.OwnedRepository
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	if m.Checkout.Specification.Branch == r.IntegrationBranch {
		return fmt.Errorf("integration branch must remain unchecked out; use an item implementation branch")
	}
	return nil
}
